package backup

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

// Scheduled S3 backups are configured in the admin UI and stored in SQLite
// (backup_s3_settings, single row id=1). The secret access key is sealed with
// SECRET_ENCRYPTION_KEY like the other stored secrets.

const (
	DefaultS3Interval = 24 * time.Hour
	MinS3Interval     = time.Minute
)

var (
	// ErrS3KeyMissing is returned when saving a secret access key requires SECRET_ENCRYPTION_KEY but it is unset.
	ErrS3KeyMissing = errors.New("SECRET_ENCRYPTION_KEY is not set or invalid")
	// ErrInvalidS3Settings wraps every validation failure; the message says which field.
	ErrInvalidS3Settings = errors.New("invalid S3 backup settings")
	// ErrS3NotConfigured is returned when a test or run needs a bucket that is not saved yet.
	ErrS3NotConfigured = errors.New("S3 backups are not configured")
)

func invalidS3(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidS3Settings, fmt.Sprintf(format, args...))
}

// S3SettingsStore persists scheduled S3 backup settings.
type S3SettingsStore struct {
	db  *sql.DB
	key []byte // 32 bytes from env; may be nil
}

// NewS3SettingsStore creates a store. key may be nil if SECRET_ENCRYPTION_KEY is not set.
func NewS3SettingsStore(db *sql.DB, key []byte) *S3SettingsStore {
	return &S3SettingsStore{db: db, key: key}
}

// KeyConfigured reports whether a 32-byte encryption key is available.
func (s *S3SettingsStore) KeyConfigured() bool {
	return len(s.key) == 32
}

// PublicS3Settings is safe to return to clients (no secret).
type PublicS3Settings struct {
	EncryptionKeySet bool   `json:"encryption_key_set"`
	Enabled          bool   `json:"enabled"`
	Configured       bool   `json:"configured"`
	Bucket           string `json:"bucket"`
	Region           string `json:"region"`
	KeyPrefix        string `json:"key_prefix"`
	Endpoint         string `json:"endpoint"`
	UsePathStyle     bool   `json:"use_path_style"`
	AccessKeyID      string `json:"access_key_id"`
	SecretConfigured bool   `json:"secret_configured"`
	Interval         string `json:"interval"`
	RetentionMax     int    `json:"retention_max"`
	RetentionDays    int    `json:"retention_days"`
}

// SaveS3Input is the admin form. An empty SecretAccessKey keeps the stored one;
// an empty AccessKeyID clears both, so the AWS default credential chain is used.
type SaveS3Input struct {
	Enabled         bool
	Bucket          string
	Region          string
	KeyPrefix       string
	Endpoint        string
	UsePathStyle    bool
	AccessKeyID     string
	SecretAccessKey string
	Interval        string
	RetentionMax    int
	RetentionDays   int
}

// s3Row mirrors the table. It is also the restore carry-over format, so its
// JSON keys must stay stable.
type s3Row struct {
	Enabled       bool   `json:"enabled"`
	Bucket        string `json:"bucket"`
	Region        string `json:"region"`
	KeyPrefix     string `json:"key_prefix"`
	Endpoint      string `json:"endpoint"`
	UsePathStyle  bool   `json:"use_path_style"`
	AccessKeyID   string `json:"access_key_id"`
	SecretEnc     []byte `json:"secret_access_key_enc,omitempty"`
	Interval      string `json:"interval"`
	RetentionMax  int    `json:"retention_max"`
	RetentionDays int    `json:"retention_days"`
}

func defaultS3Row() s3Row {
	return s3Row{Interval: DefaultS3Interval.String(), RetentionMax: 14, RetentionDays: 30}
}

func loadS3Row(db *sql.DB) (*s3Row, error) {
	var r s3Row
	var enabled, pathStyle int
	err := db.QueryRow(`
		SELECT enabled, bucket, region, key_prefix, endpoint, use_path_style,
			access_key_id, secret_access_key_enc, interval, retention_max, retention_days
		FROM backup_s3_settings WHERE id = 1
	`).Scan(&enabled, &r.Bucket, &r.Region, &r.KeyPrefix, &r.Endpoint, &pathStyle,
		&r.AccessKeyID, &r.SecretEnc, &r.Interval, &r.RetentionMax, &r.RetentionDays)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = enabled != 0
	r.UsePathStyle = pathStyle != 0
	return &r, nil
}

func writeS3Row(db *sql.DB, r s3Row) error {
	_, err := db.Exec(`
		INSERT INTO backup_s3_settings (
			id, enabled, bucket, region, key_prefix, endpoint, use_path_style,
			access_key_id, secret_access_key_enc, interval, retention_max, retention_days, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled, bucket = excluded.bucket, region = excluded.region,
			key_prefix = excluded.key_prefix, endpoint = excluded.endpoint,
			use_path_style = excluded.use_path_style, access_key_id = excluded.access_key_id,
			secret_access_key_enc = excluded.secret_access_key_enc, interval = excluded.interval,
			retention_max = excluded.retention_max, retention_days = excluded.retention_days,
			updated_at = excluded.updated_at
	`, boolInt(r.Enabled), r.Bucket, r.Region, r.KeyPrefix, r.Endpoint, boolInt(r.UsePathStyle),
		r.AccessKeyID, r.SecretEnc, r.Interval, r.RetentionMax, r.RetentionDays,
		time.Now().UTC().Format(time.RFC3339))
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// GetPublic returns the saved settings for the admin UI.
func (s *S3SettingsStore) GetPublic() (*PublicS3Settings, error) {
	r, err := loadS3Row(s.db)
	if err != nil {
		return nil, err
	}
	if r == nil {
		d := defaultS3Row()
		r = &d
	}
	return &PublicS3Settings{
		EncryptionKeySet: s.KeyConfigured(),
		Enabled:          r.Enabled,
		Configured:       r.Bucket != "",
		Bucket:           r.Bucket,
		Region:           r.Region,
		KeyPrefix:        r.KeyPrefix,
		Endpoint:         r.Endpoint,
		UsePathStyle:     r.UsePathStyle,
		AccessKeyID:      r.AccessKeyID,
		SecretConfigured: len(r.SecretEnc) > 0,
		Interval:         r.Interval,
		RetentionMax:     r.RetentionMax,
		RetentionDays:    r.RetentionDays,
	}, nil
}

// Save validates and stores the settings.
func (s *S3SettingsStore) Save(in SaveS3Input) error {
	stored, err := loadS3Row(s.db)
	if err != nil {
		return err
	}
	if stored == nil {
		d := defaultS3Row()
		stored = &d
	}

	r := s3Row{
		Enabled:       in.Enabled,
		Bucket:        strings.TrimSpace(in.Bucket),
		Region:        strings.TrimSpace(in.Region),
		KeyPrefix:     strings.Trim(strings.TrimSpace(in.KeyPrefix), "/"),
		Endpoint:      strings.TrimRight(strings.TrimSpace(in.Endpoint), "/"),
		UsePathStyle:  in.UsePathStyle,
		AccessKeyID:   strings.TrimSpace(in.AccessKeyID),
		RetentionMax:  in.RetentionMax,
		RetentionDays: in.RetentionDays,
	}

	if strings.ContainsAny(r.Bucket, " /\t") {
		return invalidS3("bucket must be a bucket name, not a path or URL")
	}
	if r.Enabled && r.Bucket == "" {
		return invalidS3("bucket is required when backups are on")
	}
	if r.Endpoint != "" {
		u, err := url.Parse(r.Endpoint)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return invalidS3("endpoint must be a valid http(s) URL")
		}
	}

	interval := strings.TrimSpace(in.Interval)
	if interval == "" {
		interval = DefaultS3Interval.String()
	}
	d, err := time.ParseDuration(interval)
	if err != nil {
		return invalidS3("interval must be a duration such as 24h or 6h")
	}
	if d < MinS3Interval {
		return invalidS3("interval must be at least %s", MinS3Interval)
	}
	r.Interval = d.String()

	if r.RetentionMax < 0 || r.RetentionDays < 0 {
		return invalidS3("retention values must be zero or positive")
	}

	secret := strings.TrimSpace(in.SecretAccessKey)
	switch {
	case r.AccessKeyID == "" && secret != "":
		return invalidS3("enter the access key ID that belongs to the secret access key")
	case r.AccessKeyID == "":
		// Default credential chain (IAM role, AWS_* env, shared config): no stored secret.
		r.SecretEnc = nil
	case secret != "":
		if !s.KeyConfigured() {
			return ErrS3KeyMissing
		}
		r.SecretEnc, err = secrets.Seal(s.key, []byte(secret))
		if err != nil {
			return fmt.Errorf("encrypt s3 secret access key: %w", err)
		}
	case r.AccessKeyID != stored.AccessKeyID || len(stored.SecretEnc) == 0:
		// A kept secret belongs to the old key ID, so a new ID needs its own secret.
		return invalidS3("enter the secret access key for this access key ID")
	default:
		r.SecretEnc = stored.SecretEnc
	}

	return writeS3Row(s.db, r)
}

// LoadConfig returns the saved settings as scheduler config, with the secret decrypted.
// A missing row yields a disabled config.
func (s *S3SettingsStore) LoadConfig() (S3SchedulerConfig, error) {
	r, err := loadS3Row(s.db)
	if err != nil || r == nil {
		return S3SchedulerConfig{}, err
	}
	cfg := S3SchedulerConfig{
		Enabled:       r.Enabled,
		Bucket:        r.Bucket,
		Region:        r.Region,
		RootPrefix:    r.KeyPrefix,
		Endpoint:      r.Endpoint,
		UsePathStyle:  r.UsePathStyle,
		StaticKey:     r.AccessKeyID,
		RetentionMax:  r.RetentionMax,
		RetentionDays: r.RetentionDays,
	}
	cfg.Interval, err = time.ParseDuration(r.Interval)
	if err != nil || cfg.Interval < MinS3Interval {
		cfg.Interval = DefaultS3Interval
	}
	if len(r.SecretEnc) > 0 {
		if !s.KeyConfigured() {
			return S3SchedulerConfig{}, fmt.Errorf("decrypt s3 secret access key: %w", ErrS3KeyMissing)
		}
		raw, err := secrets.Open(s.key, r.SecretEnc)
		if err != nil {
			return S3SchedulerConfig{}, fmt.Errorf("decrypt s3 secret access key (wrong SECRET_ENCRYPTION_KEY?): %w", err)
		}
		cfg.StaticSecret = string(raw)
	}
	return cfg, nil
}

// A restore replaces app.db, which holds these settings. The running instance's
// settings are kept instead of the archive's: a restore brings back data, not
// where backups go. StageImportApply writes the current row next to the staged
// database; ApplyPendingRestoreAtStartup moves it into the data directory; after
// migrations, ApplyCarriedS3Settings writes it into the restored database.
const (
	stagedS3SettingsFile  = "backup_s3_settings.json"
	carriedS3SettingsFile = "_restore_backup_s3_settings.json"
)

type carriedS3Settings struct {
	Present bool   `json:"present"`
	Row     *s3Row `json:"row,omitempty"`
}

func writeS3SettingsCarryOver(db *sql.DB, path string) error {
	r, err := loadS3Row(db)
	if err != nil {
		return fmt.Errorf("read s3 backup settings: %w", err)
	}
	b, err := json.Marshal(carriedS3Settings{Present: r != nil, Row: r})
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

// ApplyCarriedS3Settings writes the settings kept across a restore into db and
// removes the carry-over file. It does nothing when no restore was applied.
func ApplyCarriedS3Settings(db *sql.DB, dataDir string) error {
	path := filepath.Join(dataDir, carriedS3SettingsFile)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var c carriedS3Settings
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("parse %s: %w", carriedS3SettingsFile, err)
	}
	if c.Present && c.Row != nil {
		err = writeS3Row(db, *c.Row)
	} else {
		_, err = db.Exec(`DELETE FROM backup_s3_settings WHERE id = 1`)
	}
	if err != nil {
		return fmt.Errorf("restore s3 backup settings: %w", err)
	}
	if err := os.Remove(path); err != nil {
		log.Printf("WARNING: s3 backup settings restored, but %s could not be removed: %v", carriedS3SettingsFile, err)
	}
	return nil
}
