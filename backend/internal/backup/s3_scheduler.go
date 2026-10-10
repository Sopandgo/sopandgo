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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	automatedKeySegment = "automated/"
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
	p := strings.Trim(rootPrefix, "/")
	if p == "" {
		return automatedKeySegment
	}
	return p + "/" + automatedKeySegment
}

// auditLogger is satisfied by *audit.Logger.
type auditLogger interface {
	Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
}

// s3Target is everything one run needs, snapshotted so a concurrent Configure
// cannot change the bucket halfway through an upload.
type s3Target struct {
	client     *s3.Client
	cfg        S3SchedulerConfig
	listPrefix string
}

func newS3Client(ctx context.Context, cfg S3SchedulerConfig) (*s3.Client, error) {
	opts := []func(*config.LoadOptions) error{}
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}
	if cfg.StaticKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.StaticKey, cfg.StaticSecret, ""),
		))
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		ep := cfg.Endpoint
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(ep)
			o.UsePathStyle = cfg.UsePathStyle
		})
	} else if cfg.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}
	return s3.NewFromConfig(awsCfg, s3Opts...), nil
}

// S3Scheduler runs periodic uploads of the standard backup zip to S3. It always
// exists; Configure switches it on, off, or to new settings without a restart.
type S3Scheduler struct {
	backupSvc   *Service
	auditLogger auditLogger
	onFailure   func(ctx context.Context, err error)

	runMu sync.Mutex // one upload at a time (timer and RunNow)
	wake  chan struct{}

	mu            sync.Mutex
	target        s3Target
	lastRun       time.Time
	lastSuccess   time.Time
	lastObjectKey string
	lastError     string
	nextRun       time.Time
}

// NewS3Scheduler returns a scheduler that is off until Configure enables it.
// auditLogger may be nil (no audit entries).
func NewS3Scheduler(backupSvc *Service, auditLogger auditLogger) *S3Scheduler {
	return &S3Scheduler{
		backupSvc:   backupSvc,
		auditLogger: auditLogger,
		wake:        make(chan struct{}, 1),
	}
}

// SetOnFailure registers an optional callback invoked after a failed automated backup
// (in addition to the audit log entry). Safe to call before Start.
func (sch *S3Scheduler) SetOnFailure(fn func(ctx context.Context, err error)) {
	sch.onFailure = fn
}

// Configure applies cfg. A disabled cfg stops scheduled runs. Keeping the same
// interval keeps the next run time; otherwise the next run is one interval away.
func (sch *S3Scheduler) Configure(ctx context.Context, cfg S3SchedulerConfig) error {
	var client *s3.Client
	if cfg.Enabled {
		c, err := newS3Client(ctx, cfg)
		if err != nil {
			return err
		}
		client = c
	}

	sch.mu.Lock()
	prev := sch.target.cfg
	sch.target = s3Target{client: client, cfg: cfg, listPrefix: automatedListPrefix(cfg.RootPrefix)}
	switch {
	case !cfg.Enabled:
		sch.nextRun = time.Time{}
	case prev.Enabled && prev.Interval == cfg.Interval && !sch.nextRun.IsZero():
	default:
		sch.nextRun = time.Now().UTC().Add(cfg.Interval)
	}
	sch.mu.Unlock()

	sch.signal()
	return nil
}

func (sch *S3Scheduler) signal() {
	select {
	case sch.wake <- struct{}{}:
	default:
	}
}

// Status returns the latest scheduler snapshot.
func (sch *S3Scheduler) Status() S3SchedulerStatus {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	cfg := sch.target.cfg
	st := S3SchedulerStatus{
		Enabled:       cfg.Enabled,
		LastError:     sch.lastError,
		LastObjectKey: sch.lastObjectKey,
	}
	if cfg.Enabled {
		st.Bucket = cfg.Bucket
		st.KeyPrefix = sch.target.listPrefix
		st.Interval = cfg.Interval.String()
		st.RetentionMax = cfg.RetentionMax
		st.RetentionDays = cfg.RetentionDays
	}
	if !sch.lastRun.IsZero() {
		st.LastRunUTC = sch.lastRun.UTC().Format(time.RFC3339)
	}
	if !sch.lastSuccess.IsZero() {
		st.LastSuccessUTC = sch.lastSuccess.UTC().Format(time.RFC3339)
	}
	if !sch.nextRun.IsZero() {
		st.NextRunUTC = sch.nextRun.UTC().Format(time.RFC3339)
	}
	return st
}

// Start runs scheduled backups until ctx is done, following Configure calls.
func (sch *S3Scheduler) Start(ctx context.Context) {
	go func() {
		for {
			sch.mu.Lock()
			next := sch.nextRun
			sch.mu.Unlock()

			var due <-chan time.Time
			var timer *time.Timer
			if !next.IsZero() {
				timer = time.NewTimer(time.Until(next))
				due = timer.C
			}
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case <-sch.wake:
				if timer != nil {
					timer.Stop()
				}
			case <-due:
				_, _ = sch.run(ctx)
			}
		}
	}()
}

// RunNow uploads one backup with the saved settings and moves the next scheduled
// run one interval out. It returns the object key.
func (sch *S3Scheduler) RunNow(ctx context.Context) (string, error) {
	key, err := sch.run(ctx)
	sch.signal()
	return key, err
}

func (sch *S3Scheduler) run(ctx context.Context) (string, error) {
	sch.runMu.Lock()
	defer sch.runMu.Unlock()

	sch.mu.Lock()
	t := sch.target
	sch.mu.Unlock()
	if !t.cfg.Enabled || t.client == nil {
		return "", ErrS3NotEnabled
	}

	runAt := time.Now().UTC()
	objKey, manifest, err := uploadBackup(ctx, sch.backupSvc, t, runAt)

	sch.mu.Lock()
	sch.lastRun = runAt
	if sch.target.cfg.Enabled {
		sch.nextRun = runAt.Add(sch.target.cfg.Interval)
	}
	if err != nil {
		sch.lastError = err.Error()
	} else {
		sch.lastError = ""
		sch.lastObjectKey = objKey
		sch.lastSuccess = runAt
	}
	sch.mu.Unlock()

	if err != nil {
		log.Printf("s3 automatic backup failed: %v", err)
		if sch.auditLogger != nil {
			if auditErr := sch.auditLogger.Log(nil, audit.EventBackupS3Failed, audit.EntitySystem, "backup", nil, map[string]any{
				"error": err.Error(),
			}); auditErr != nil {
				log.Printf("audit write failed for S3 backup failure: %v", auditErr)
			}
		}
		if sch.onFailure != nil {
			sch.onFailure(ctx, err)
		}
		return "", err
	}

	log.Printf("s3 automatic backup uploaded: %s", objKey)
	if sch.auditLogger != nil {
		if auditErr := sch.auditLogger.Log(nil, audit.EventBackupS3Uploaded, audit.EntitySystem, "backup", nil, map[string]any{
			"bucket":     t.cfg.Bucket,
			"object_key": objKey,
			"manifest":   manifest,
		}); auditErr != nil {
			log.Printf("audit write failed for S3 backup upload: %v", auditErr)
		}
	}
	return objKey, nil
}

func uploadBackup(ctx context.Context, svc *Service, t s3Target, runAt time.Time) (string, Manifest, error) {
	zipPath, _, manifest, err := svc.ExportZip()
	if err != nil {
		return "", Manifest{}, fmt.Errorf("export zip: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(zipPath))

	key := objectKeyForRun(t.listPrefix, runAt)

	f, err := os.Open(zipPath)
	if err != nil {
		return "", manifest, fmt.Errorf("open zip: %w", err)
	}
	defer f.Close()

	_, err = t.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(t.cfg.Bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String("application/zip"),
	})
	if err != nil {
		return "", manifest, fmt.Errorf("s3 PutObject: %w", err)
	}

	if err := pruneOldObjects(ctx, t); err != nil {
		return "", manifest, fmt.Errorf("retention prune: %w", err)
	}
	return key, manifest, nil
}

// CheckS3Connection checks that cfg can write and delete under the backup prefix:
// it uploads a small probe object next to the automated archives and removes it.
// The probe name never matches the retention filter.
func CheckS3Connection(ctx context.Context, cfg S3SchedulerConfig) error {
	if cfg.Bucket == "" {
		return ErrS3NotConfigured
	}
	client, err := newS3Client(ctx, cfg)
	if err != nil {
		return err
	}
	key := automatedListPrefix(cfg.RootPrefix) + connectionTestName
	if _, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader("sopandgo connection test\n"),
		ContentType: aws.String("text/plain"),
	}); err != nil {
		return fmt.Errorf("s3 PutObject: %w", err)
	}
	if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("s3 DeleteObject: %w", err)
	}
	return nil
}

func objectKeyForRun(listPrefix string, t time.Time) string {
	name := fmt.Sprintf("sopandgo-automated-%s.zip", t.UTC().Format("20060102T150405"))
	return listPrefix + name
}

type s3ObjectMeta struct {
	key string
	t   time.Time
}

func listAutomatedObjects(ctx context.Context, t s3Target) ([]s3ObjectMeta, error) {
	var out []s3ObjectMeta
	paginator := s3.NewListObjectsV2Paginator(t.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(t.cfg.Bucket),
		Prefix: aws.String(t.listPrefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || obj.LastModified == nil {
				continue
			}
			if !strings.HasSuffix(*obj.Key, ".zip") {
				continue
			}
			base := path.Base(*obj.Key)
			if !strings.HasPrefix(base, "sopandgo-automated-") {
				continue
			}
			out = append(out, s3ObjectMeta{key: *obj.Key, t: *obj.LastModified})
		}
	}
	return out, nil
}

// keysToDelete returns object keys to remove given retention rules.
// objs must be sorted by time ascending (oldest first).
func keysToDeleteForRetention(objs []s3ObjectMeta, maxKeep int, retentionDays int, now time.Time) []string {
	if len(objs) == 0 {
		return nil
	}
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].t.Before(objs[j].t)
	})
	var toDelete []string
	var survivors []s3ObjectMeta
	cutoff := time.Time{}
	if retentionDays > 0 {
		cutoff = now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	}
	for _, o := range objs {
		if retentionDays > 0 && o.t.Before(cutoff) {
			toDelete = append(toDelete, o.key)
			continue
		}
		survivors = append(survivors, o)
	}
	if maxKeep > 0 && len(survivors) > maxKeep {
		excess := len(survivors) - maxKeep
		for i := 0; i < excess; i++ {
			toDelete = append(toDelete, survivors[i].key)
		}
	}
	return toDelete
}

func pruneOldObjects(ctx context.Context, t s3Target) error {
	objs, err := listAutomatedObjects(ctx, t)
	if err != nil {
		return err
	}
	toDel := keysToDeleteForRetention(objs, t.cfg.RetentionMax, t.cfg.RetentionDays, time.Now().UTC())
	for _, key := range toDel {
		_, err := t.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(t.cfg.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("delete %q: %w", key, err)
		}
	}
	if len(toDel) > 0 {
		log.Printf("s3 automatic backup retention: deleted %d object(s)", len(toDel))
	}
	return nil
}
