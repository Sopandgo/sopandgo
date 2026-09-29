package backup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const automatedKeySegment = "automated/"

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

// S3SchedulerConfig is loaded from environment variables.
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
	StaticKey     string
	StaticSecret  string
}

// LoadS3SchedulerConfigFromEnv parses S3 backup settings. When disabled, returns zero config and nil error.
func LoadS3SchedulerConfigFromEnv() (S3SchedulerConfig, error) {
	enabled := parseBoolEnv("BACKUP_S3_ENABLED")
	if !enabled {
		return S3SchedulerConfig{}, nil
	}
	cfg := S3SchedulerConfig{
		Enabled:      true,
		Bucket:       strings.TrimSpace(os.Getenv("BACKUP_S3_BUCKET")),
		Region:       strings.TrimSpace(os.Getenv("BACKUP_S3_REGION")),
		RootPrefix:   strings.Trim(strings.TrimSpace(os.Getenv("BACKUP_S3_PREFIX")), "/"),
		Endpoint:     strings.TrimSpace(os.Getenv("BACKUP_S3_ENDPOINT")),
		UsePathStyle: parseBoolEnv("BACKUP_S3_USE_PATH_STYLE"),
		StaticKey:    strings.TrimSpace(os.Getenv("BACKUP_S3_ACCESS_KEY_ID")),
		StaticSecret: strings.TrimSpace(os.Getenv("BACKUP_S3_SECRET_ACCESS_KEY")),
	}
	if cfg.Bucket == "" {
		return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_ENABLED is true but BACKUP_S3_BUCKET is empty")
	}
	intervalStr := strings.TrimSpace(os.Getenv("BACKUP_S3_INTERVAL"))
	if intervalStr == "" {
		intervalStr = "24h"
	}
	d, err := time.ParseDuration(intervalStr)
	if err != nil {
		return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_INTERVAL: %w", err)
	}
	if d < time.Minute {
		return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_INTERVAL must be at least 1m")
	}
	cfg.Interval = d

	if v := strings.TrimSpace(os.Getenv("BACKUP_S3_RETENTION_MAX")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_RETENTION_MAX must be a non-negative integer")
		}
		cfg.RetentionMax = n
	}
	if v := strings.TrimSpace(os.Getenv("BACKUP_S3_RETENTION_DAYS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_RETENTION_DAYS must be a non-negative integer")
		}
		cfg.RetentionDays = n
	}
	if cfg.RetentionMax == 0 && cfg.RetentionDays == 0 {
		log.Println("WARNING: S3 automatic backups enabled but both BACKUP_S3_RETENTION_MAX and BACKUP_S3_RETENTION_DAYS are unset or zero; old objects are never pruned")
	}
	if cfg.StaticKey != "" && cfg.StaticSecret == "" || cfg.StaticKey == "" && cfg.StaticSecret != "" {
		return S3SchedulerConfig{}, fmt.Errorf("BACKUP_S3_ACCESS_KEY_ID and BACKUP_S3_SECRET_ACCESS_KEY must both be set or both empty")
	}
	return cfg, nil
}

func parseBoolEnv(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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

// S3Scheduler runs periodic uploads of the standard backup zip to S3.
type S3Scheduler struct {
	cfg         S3SchedulerConfig
	backupSvc   *Service
	auditLogger auditLogger
	s3Client    *s3.Client
	listPrefix  string
	onFailure   func(ctx context.Context, err error)

	mu            sync.Mutex
	lastRun       time.Time
	lastSuccess   time.Time
	lastObjectKey string
	lastError     string
	nextRun       time.Time
}

// NewS3Scheduler builds an S3 client and scheduler. ctx is used only for AWS config resolution.
// auditLogger may be nil (no audit entries).
func NewS3Scheduler(ctx context.Context, cfg S3SchedulerConfig, backupSvc *Service, auditLogger auditLogger) (*S3Scheduler, error) {
	if !cfg.Enabled {
		return nil, nil
	}
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

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	return &S3Scheduler{
		cfg:         cfg,
		backupSvc:   backupSvc,
		auditLogger: auditLogger,
		s3Client:    client,
		listPrefix:  automatedListPrefix(cfg.RootPrefix),
	}, nil
}

// SetOnFailure registers an optional callback invoked after a failed automated backup
// (in addition to the audit log entry). Safe to call before Start.
func (sch *S3Scheduler) SetOnFailure(fn func(ctx context.Context, err error)) {
	if sch == nil {
		return
	}
	sch.onFailure = fn
}

// Status returns the latest scheduler snapshot.
func (sch *S3Scheduler) Status() S3SchedulerStatus {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	st := S3SchedulerStatus{
		Enabled:       true,
		Bucket:        sch.cfg.Bucket,
		KeyPrefix:     sch.listPrefix,
		Interval:      sch.cfg.Interval.String(),
		RetentionMax:  sch.cfg.RetentionMax,
		RetentionDays: sch.cfg.RetentionDays,
		LastError:     sch.lastError,
		LastObjectKey: sch.lastObjectKey,
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

func (sch *S3Scheduler) setStatusAfterRun(runTime time.Time, nextRun time.Time, objKey string, runErr error) {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	sch.lastRun = runTime
	sch.nextRun = nextRun
	if runErr != nil {
		sch.lastError = runErr.Error()
		return
	}
	sch.lastError = ""
	if objKey != "" {
		sch.lastObjectKey = objKey
	}
	sch.lastSuccess = runTime
}

// Start runs the first backup after initialDelay, then every cfg.Interval until ctx is done.
func (sch *S3Scheduler) Start(ctx context.Context, initialDelay time.Duration) {
	if sch == nil {
		return
	}
	sch.mu.Lock()
	sch.nextRun = time.Now().UTC().Add(initialDelay)
	sch.mu.Unlock()
	go func() {
		timer := time.NewTimer(initialDelay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				runAt := time.Now().UTC()
				next := runAt.Add(sch.cfg.Interval)
				objKey, manifest, err := sch.runOnce(ctx)
				sch.setStatusAfterRun(runAt, next, objKey, err)
				if err != nil {
					log.Printf("s3 automatic backup failed: %v", err)
					if sch.auditLogger != nil {
						_ = sch.auditLogger.Log(nil, audit.EventBackupS3Failed, audit.EntitySystem, "backup", nil, map[string]any{
							"error": err.Error(),
						})
					}
					if sch.onFailure != nil {
						sch.onFailure(ctx, err)
					}
				} else {
					log.Printf("s3 automatic backup uploaded: %s", objKey)
					if sch.auditLogger != nil {
						_ = sch.auditLogger.Log(nil, audit.EventBackupS3Uploaded, audit.EntitySystem, "backup", nil, map[string]any{
							"bucket":     sch.cfg.Bucket,
							"object_key": objKey,
							"manifest":   manifest,
						})
					}
				}
				timer.Reset(sch.cfg.Interval)
			}
		}
	}()
}

func (sch *S3Scheduler) runOnce(ctx context.Context) (string, Manifest, error) {
	zipPath, _, manifest, err := sch.backupSvc.ExportZip()
	if err != nil {
		return "", Manifest{}, fmt.Errorf("export zip: %w", err)
	}
	zipDir := filepath.Dir(zipPath)
	defer os.RemoveAll(zipDir)

	key := sch.objectKeyForRun(time.Now().UTC())

	f, err := os.Open(zipPath)
	if err != nil {
		return "", manifest, fmt.Errorf("open zip: %w", err)
	}
	defer f.Close()

	_, err = sch.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(sch.cfg.Bucket),
		Key:         aws.String(key),
		Body:        f,
		ContentType: aws.String("application/zip"),
	})
	if err != nil {
		return "", manifest, fmt.Errorf("s3 PutObject: %w", err)
	}

	if err := sch.pruneOldObjects(ctx); err != nil {
		return "", manifest, fmt.Errorf("retention prune: %w", err)
	}

	return key, manifest, nil
}

func (sch *S3Scheduler) objectKeyForRun(t time.Time) string {
	name := fmt.Sprintf("sopandgo-automated-%s.zip", t.UTC().Format("20060102T150405"))
	return sch.listPrefix + name
}

type s3ObjectMeta struct {
	key string
	t   time.Time
}

func (sch *S3Scheduler) listAutomatedObjects(ctx context.Context) ([]s3ObjectMeta, error) {
	var out []s3ObjectMeta
	paginator := s3.NewListObjectsV2Paginator(sch.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(sch.cfg.Bucket),
		Prefix: aws.String(sch.listPrefix),
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

func (sch *S3Scheduler) pruneOldObjects(ctx context.Context) error {
	objs, err := sch.listAutomatedObjects(ctx)
	if err != nil {
		return err
	}
	toDel := keysToDeleteForRetention(objs, sch.cfg.RetentionMax, sch.cfg.RetentionDays, time.Now().UTC())
	for _, key := range toDel {
		_, err := sch.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(sch.cfg.Bucket),
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

// RunOnceImmediately exports and uploads once (e.g. tests). Uses ctx for S3 calls.
func (sch *S3Scheduler) RunOnceImmediately(ctx context.Context) error {
	if sch == nil {
		return nil
	}
	objKey, _, err := sch.runOnce(ctx)
	now := time.Now().UTC()
	sch.setStatusAfterRun(now, now.Add(sch.cfg.Interval), objKey, err)
	return err
}

// TestHookS3Client replaces the S3 client (for tests).
func (sch *S3Scheduler) TestHookS3Client(c *s3.Client) {
	if sch == nil {
		return
	}
	sch.s3Client = c
}

func (sch *S3Scheduler) TestHookListPrefix(prefix string) {
	if sch == nil {
		return
	}
	sch.listPrefix = prefix
}
