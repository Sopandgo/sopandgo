package backup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/objectstore"
	"github.com/sopandgo/sopandgo/backend/internal/schedule"
)

const (
	automatedKeySegment = "automated"
	connectionTestName  = "sopandgo-connection-test.txt"
)

// ErrS3NotEnabled is returned by RunNow when scheduled backups are off.
var ErrS3NotEnabled = errors.New("S3 backups are off")

// S3SchedulerStatus is a snapshot for admin status API (no secrets).
type S3SchedulerStatus struct {
	Enabled        bool   `json:"enabled"`
	Bucket         string `json:"bucket,omitempty"`
	KeyPrefix      string `json:"key_prefix,omitempty"`
	Interval       string `json:"interval,omitempty"`
	RetentionMax   int    `json:"retention_max_objects,omitempty"`
	RetentionDays  int    `json:"retention_days,omitempty"`
	LastRunUTC     string `json:"last_run_utc,omitempty"`
	LastSuccessUTC string `json:"last_success_utc,omitempty"`
	LastObjectKey  string `json:"last_object_key,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	NextRunUTC     string `json:"next_run_utc,omitempty"`
}

// S3SchedulerConfig is the saved S3 backup settings (S3SettingsStore.LoadConfig).
type S3SchedulerConfig struct {
	Enabled       bool
	Bucket        string
	Region        string
	RootPrefix    string // trimmed user prefix; automated keys live under rootPrefix/automated/
	Interval      time.Duration
	RetentionMax  int    // 0 = no count limit
	RetentionDays int    // 0 = no age limit
	Endpoint      string // optional, e.g. MinIO
	UsePathStyle  bool
	StaticKey     string // empty = AWS default credential chain
	StaticSecret  string
}

func automatedListPrefix(rootPrefix string) string {
	return objectstore.Join(strings.Trim(rootPrefix, "/"), automatedKeySegment)
}

// auditLogger is satisfied by *audit.Logger.
type auditLogger interface {
	Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
}

// S3Scheduler is the automatic S3 backup job. It always exists; SaveSettings
// and Configure switch it on, off, or to new settings without a restart. The
// schedule package owns timing, backoff, status and failure reporting.
type S3Scheduler struct {
	backupSvc   *Service
	settings    *S3SettingsStore
	open        objectstore.Opener
	auditLogger auditLogger
	onFailure   func(ctx context.Context, err error)
	job         *schedule.Scheduled[string]

	// applyMu pairs a saved row with the Configure that follows it, so a run
	// snapshots the settings and opens the bucket from the same save.
	applyMu sync.Mutex

	mu            sync.Mutex
	cfg           S3SchedulerConfig
	lastObjectKey string
}

// NewS3Scheduler returns a job that is off until Configure enables it. Runs
// and the connection test reach the bucket through open. auditLogger may be
// nil (no audit entries).
func NewS3Scheduler(backupSvc *Service, settings *S3SettingsStore, open objectstore.Opener, auditLogger auditLogger) *S3Scheduler {
	sch := &S3Scheduler{backupSvc: backupSvc, settings: settings, open: open, auditLogger: auditLogger}
	sch.job = schedule.New(schedule.Job[string]{
		Name:            "s3 automatic backup",
		Run:             sch.run,
		FailureEvent:    audit.EventBackupS3Failed,
		FailureEntityID: "backup",
		OnFailure: func(ctx context.Context, err error) {
			if sch.onFailure != nil {
				sch.onFailure(ctx, err)
			}
		},
	}, auditLogger)
	return sch
}

// SetOnFailure registers an optional notice sent after each failed backup (in
// addition to the audit log entry). It runs on its own goroutine after the
// run is over. Safe to call before Start.
func (sch *S3Scheduler) SetOnFailure(fn func(ctx context.Context, err error)) {
	sch.onFailure = fn
}

// Configure applies cfg. A disabled cfg stops scheduled runs. Keeping the same
// interval keeps the next run time; otherwise the next run is one interval away.
// It does not contact the bucket.
func (sch *S3Scheduler) Configure(cfg S3SchedulerConfig) {
	sch.mu.Lock()
	sch.cfg = cfg
	sch.mu.Unlock()
	sch.job.Configure(cfg.Enabled, cfg.Interval)
}

// LoadSaved configures the job from the saved settings, at startup.
func (sch *S3Scheduler) LoadSaved() (S3SchedulerConfig, error) {
	cfg, err := sch.settings.LoadConfig()
	if err != nil {
		return S3SchedulerConfig{}, err
	}
	sch.Configure(cfg)
	return cfg, nil
}

// SaveSettings validates in, saves it and writes backup_s3_settings_updated
// in one transaction, then applies the saved settings to the schedule.
// Credentials are not checked here: TestConnection or the next run reports them.
func (sch *S3Scheduler) SaveSettings(in SaveS3Input, actorUserID *string) error {
	sch.applyMu.Lock()
	defer sch.applyMu.Unlock()
	row, err := sch.settings.saveAudited(in, sch.auditLogger, actorUserID)
	if err != nil {
		return err
	}
	sch.Configure(configFromRow(row))
	return nil
}

// Status returns the latest job snapshot.
func (sch *S3Scheduler) Status() S3SchedulerStatus {
	snap := sch.job.Status()
	sch.mu.Lock()
	cfg, lastKey := sch.cfg, sch.lastObjectKey
	sch.mu.Unlock()

	st := S3SchedulerStatus{
		Enabled:        snap.Enabled,
		LastError:      snap.LastError,
		LastObjectKey:  lastKey,
		LastRunUTC:     formatUTC(snap.LastAttempt),
		LastSuccessUTC: formatUTC(snap.LastSuccess),
		NextRunUTC:     formatUTC(snap.NextRun),
	}
	if snap.Enabled {
		st.Bucket = cfg.Bucket
		st.KeyPrefix = automatedListPrefix(cfg.RootPrefix)
		st.Interval = cfg.Interval.String()
		st.RetentionMax = cfg.RetentionMax
		st.RetentionDays = cfg.RetentionDays
	}
	return st
}

func formatUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// Start runs scheduled backups until ctx is done.
func (sch *S3Scheduler) Start(ctx context.Context) {
	sch.job.Start(ctx)
}

// RunNow uploads one backup with the saved settings and returns the object
// key. Success moves the next run one interval out; failure, to the retry time.
func (sch *S3Scheduler) RunNow(ctx context.Context) (string, error) {
	key, err := sch.job.RunNow(ctx)
	if errors.Is(err, schedule.ErrNotEnabled) {
		return "", ErrS3NotEnabled
	}
	return key, err
}

// run is one backup: export, upload, prune. It opens the bucket once, so the
// archive and its prune always go to the same bucket.
func (sch *S3Scheduler) run(ctx context.Context) (string, error) {
	sch.applyMu.Lock()
	sch.mu.Lock()
	cfg := sch.cfg
	sch.mu.Unlock()
	store, err := sch.open(ctx)
	sch.applyMu.Unlock()
	if err != nil {
		return "", err
	}
	objKey, manifest, err := uploadBackup(ctx, sch.backupSvc, store, cfg, time.Now().UTC())
	if err != nil {
		return "", err
	}

	sch.mu.Lock()
	sch.lastObjectKey = objKey
	sch.mu.Unlock()

	log.Printf("s3 automatic backup uploaded: %s", objKey)
	if sch.auditLogger != nil {
		if auditErr := sch.auditLogger.Log(nil, audit.EventBackupS3Uploaded, audit.EntitySystem, "backup", nil, map[string]any{
			"bucket":     cfg.Bucket,
			"object_key": objKey,
			"manifest":   manifest,
		}); auditErr != nil {
			log.Printf("audit write failed for S3 backup upload: %v", auditErr)
		}
	}
	return objKey, nil
}

func uploadBackup(ctx context.Context, svc *Service, store objectstore.Store, cfg S3SchedulerConfig, runAt time.Time) (string, Manifest, error) {
	zipPath, _, manifest, err := svc.ExportZip()
	if err != nil {
		return "", Manifest{}, fmt.Errorf("export zip: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(zipPath))

	listPrefix := automatedListPrefix(store.Root())
	key := objectKeyForRun(listPrefix, runAt)

	f, err := os.Open(zipPath)
	if err != nil {
		return "", manifest, fmt.Errorf("open zip: %w", err)
	}
	defer f.Close()

	if err := store.Put(ctx, key, f, "application/zip"); err != nil {
		return "", manifest, fmt.Errorf("s3 PutObject: %w", err)
	}
	if err := pruneOldObjects(ctx, store, listPrefix, cfg); err != nil {
		return "", manifest, fmt.Errorf("retention prune: %w", err)
	}
	return key, manifest, nil
}

// TestConnection checks that the saved settings can write and delete under
// the backup prefix: it uploads a small probe object next to the automated
// archives and removes it. The probe name never matches the retention filter.
func (sch *S3Scheduler) TestConnection(ctx context.Context) error {
	store, err := sch.open(ctx)
	if err != nil {
		return err
	}
	key := automatedListPrefix(store.Root()) + connectionTestName
	if err := store.Put(ctx, key, strings.NewReader("sopandgo connection test\n"), "text/plain"); err != nil {
		return fmt.Errorf("s3 PutObject: %w", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		return fmt.Errorf("s3 DeleteObject: %w", err)
	}
	return nil
}

func objectKeyForRun(listPrefix string, t time.Time) string {
	name := fmt.Sprintf("sopandgo-automated-%s.zip", t.UTC().Format("20060102T150405"))
	return listPrefix + name
}

// listAutomatedObjects returns the automated archives under listPrefix. Any
// other object (the connection probe, files the lab keeps there) is left out.
func listAutomatedObjects(ctx context.Context, store objectstore.Store, listPrefix string) ([]objectstore.Object, error) {
	all, err := store.List(ctx, listPrefix)
	if err != nil {
		return nil, err
	}
	var out []objectstore.Object
	for _, o := range all {
		if o.LastModified.IsZero() || !strings.HasSuffix(o.Key, ".zip") {
			continue
		}
		if !strings.HasPrefix(path.Base(o.Key), "sopandgo-automated-") {
			continue
		}
		out = append(out, o)
	}
	return out, nil
}

// keysToDeleteForRetention returns the keys to remove: first everything older
// than retentionDays, then the oldest survivors beyond maxKeep.
func keysToDeleteForRetention(objs []objectstore.Object, maxKeep int, retentionDays int, now time.Time) []string {
	if len(objs) == 0 {
		return nil
	}
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].LastModified.Before(objs[j].LastModified)
	})
	var toDelete []string
	var survivors []objectstore.Object
	cutoff := time.Time{}
	if retentionDays > 0 {
		cutoff = now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	}
	for _, o := range objs {
		if retentionDays > 0 && o.LastModified.Before(cutoff) {
			toDelete = append(toDelete, o.Key)
			continue
		}
		survivors = append(survivors, o)
	}
	if maxKeep > 0 && len(survivors) > maxKeep {
		excess := len(survivors) - maxKeep
		for i := 0; i < excess; i++ {
			toDelete = append(toDelete, survivors[i].Key)
		}
	}
	return toDelete
}

func pruneOldObjects(ctx context.Context, store objectstore.Store, listPrefix string, cfg S3SchedulerConfig) error {
	objs, err := listAutomatedObjects(ctx, store, listPrefix)
	if err != nil {
		return err
	}
	toDel := keysToDeleteForRetention(objs, cfg.RetentionMax, cfg.RetentionDays, time.Now().UTC())
	for _, key := range toDel {
		if err := store.Delete(ctx, key); err != nil {
			return fmt.Errorf("delete %q: %w", key, err)
		}
	}
	if len(toDel) > 0 {
		log.Printf("s3 automatic backup retention: deleted %d object(s)", len(toDel))
	}
	return nil
}
