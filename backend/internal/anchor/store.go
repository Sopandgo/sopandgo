package anchor

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

const (
	DefaultInterval = time.Hour
	MinInterval     = 5 * time.Minute
	MaxInterval     = 7 * 24 * time.Hour
)

var (
	// ErrKeyMissing is returned when a TSA credential must be sealed but
	// SECRET_ENCRYPTION_KEY is unset.
	ErrKeyMissing = errors.New("SECRET_ENCRYPTION_KEY is not set or invalid")
)

// settingsRow mirrors audit_anchoring_settings. It is also the restore
// carry-over format, so its JSON keys must stay stable.
type settingsRow struct {
	Enabled     bool   `json:"enabled"`
	TSAURL      string `json:"tsa_url"`
	AuthType    string `json:"auth_type"`
	Username    string `json:"username"`
	SecretEnc   []byte `json:"secret_enc,omitempty"`
	Interval    string `json:"interval"`
	PinnedCAPEM string `json:"pinned_ca_pem"`
	InstanceID  string `json:"instance_id"`
}

// Settings is the decrypted configuration the service runs with.
type Settings struct {
	Enabled     bool
	TSAURL      string
	AuthType    string
	Username    string
	Secret      string
	Interval    time.Duration
	PinnedCAPEM string
	InstanceID  string
}

// Anchor is one stored receipt.
type Anchor struct {
	ID          int64     `json:"id"`
	EventID     int64     `json:"event_id"`
	TipHash     string    `json:"tip_hash"`
	HashVersion int       `json:"hash_version"`
	TSR         []byte    `json:"-"`
	GenTime     time.Time `json:"gen_time"`
	TSAURL      string    `json:"tsa_url"`
	TSACAPEM    string    `json:"-"`
	S3Key       string    `json:"s3_key"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store persists anchoring settings and receipts.
type Store struct {
	db  *sql.DB
	key []byte // 32 bytes from env; may be nil
}

func NewStore(db *sql.DB, key []byte) *Store {
	return &Store{db: db, key: key}
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) keyConfigured() bool { return len(s.key) == 32 }

func loadRow(db *sql.DB) (*settingsRow, error) {
	var r settingsRow
	var enabled int
	err := db.QueryRow(`
		SELECT enabled, tsa_url, auth_type, username, secret_enc, interval, pinned_ca_pem, instance_id
		FROM audit_anchoring_settings WHERE id = 1
	`).Scan(&enabled, &r.TSAURL, &r.AuthType, &r.Username, &r.SecretEnc, &r.Interval, &r.PinnedCAPEM, &r.InstanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = enabled == 1
	return &r, nil
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func writeRow(db execer, r settingsRow) error {
	enabled := 0
	if r.Enabled {
		enabled = 1
	}
	_, err := db.Exec(`
		INSERT INTO audit_anchoring_settings
			(id, enabled, tsa_url, auth_type, username, secret_enc, interval, pinned_ca_pem, instance_id, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			enabled = excluded.enabled,
			tsa_url = excluded.tsa_url,
			auth_type = excluded.auth_type,
			username = excluded.username,
			secret_enc = excluded.secret_enc,
			interval = excluded.interval,
			pinned_ca_pem = excluded.pinned_ca_pem,
			instance_id = excluded.instance_id,
			updated_at = excluded.updated_at
	`, enabled, r.TSAURL, r.AuthType, r.Username, r.SecretEnc, r.Interval, r.PinnedCAPEM, r.InstanceID,
		time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// Load returns the saved settings with the TSA secret decrypted. A missing row
// is a disabled configuration.
func (s *Store) Load() (Settings, error) {
	r, err := loadRow(s.db)
	if err != nil || r == nil {
		return Settings{AuthType: AuthNone, Interval: DefaultInterval}, err
	}
	return s.decode(*r)
}

func (s *Store) decode(r settingsRow) (Settings, error) {
	st := Settings{
		Enabled:     r.Enabled,
		TSAURL:      r.TSAURL,
		AuthType:    r.AuthType,
		Username:    r.Username,
		PinnedCAPEM: r.PinnedCAPEM,
		InstanceID:  r.InstanceID,
	}
	if st.AuthType == "" {
		st.AuthType = AuthNone
	}
	st.Interval, _ = time.ParseDuration(r.Interval)
	if st.Interval < MinInterval || st.Interval > MaxInterval {
		st.Interval = DefaultInterval
	}
	if len(r.SecretEnc) > 0 {
		if !s.keyConfigured() {
			return Settings{}, fmt.Errorf("decrypt TSA credential: %w", ErrKeyMissing)
		}
		raw, err := secrets.Open(s.key, r.SecretEnc)
		if err != nil {
			return Settings{}, fmt.Errorf("decrypt TSA credential (wrong SECRET_ENCRYPTION_KEY?): %w", err)
		}
		st.Secret = string(raw)
	}
	return st, nil
}

// save writes st inside tx, sealing the secret.
func (s *Store) save(tx execer, st Settings) error {
	r := settingsRow{
		Enabled:     st.Enabled,
		TSAURL:      st.TSAURL,
		AuthType:    st.AuthType,
		Username:    st.Username,
		Interval:    st.Interval.String(),
		PinnedCAPEM: st.PinnedCAPEM,
		InstanceID:  st.InstanceID,
	}
	if st.Secret != "" {
		if !s.keyConfigured() {
			return ErrKeyMissing
		}
		enc, err := secrets.Seal(s.key, []byte(st.Secret))
		if err != nil {
			return err
		}
		r.SecretEnc = enc
	}
	return writeRow(tx, r)
}

func newInstanceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func scanAnchor(scan func(dest ...any) error) (Anchor, error) {
	var a Anchor
	var gen, created string
	if err := scan(&a.ID, &a.EventID, &a.TipHash, &a.HashVersion, &a.TSR, &gen, &a.TSAURL, &a.TSACAPEM, &a.S3Key, &created); err != nil {
		return Anchor{}, err
	}
	a.GenTime, _ = time.Parse(time.RFC3339Nano, gen)
	a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return a, nil
}

const anchorColumns = `id, event_id, tip_hash, hash_version, tsr, gen_time, tsa_url, tsa_ca_pem, s3_key, created_at`

// Latest returns the newest anchor, or nil when there is none.
func (s *Store) Latest() (*Anchor, error) {
	a, err := scanAnchor(s.db.QueryRow(`SELECT ` + anchorColumns + ` FROM audit_anchors ORDER BY id DESC LIMIT 1`).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// List returns every anchor in creation order.
func (s *Store) List() ([]Anchor, error) {
	return s.query(`SELECT ` + anchorColumns + ` FROM audit_anchors ORDER BY id ASC`)
}

// Count returns how many anchors are stored and how many await upload.
func (s *Store) Count() (total, pending int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN s3_key = '' THEN 1 ELSE 0 END), 0) FROM audit_anchors`).Scan(&total, &pending)
	return
}

// pendingUploads returns anchors not yet copied to S3.
func (s *Store) pendingUploads() ([]Anchor, error) {
	return s.query(`SELECT ` + anchorColumns + ` FROM audit_anchors WHERE s3_key = '' ORDER BY id ASC`)
}

func (s *Store) query(q string, args ...any) ([]Anchor, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Anchor
	for rows.Next() {
		a, err := scanAnchor(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) insert(a Anchor) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO audit_anchors (event_id, tip_hash, hash_version, tsr, gen_time, tsa_url, tsa_ca_pem, s3_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.EventID, a.TipHash, a.HashVersion, a.TSR, a.GenTime.UTC().Format(time.RFC3339Nano), a.TSAURL, a.TSACAPEM, a.S3Key,
		time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) markUploaded(id int64, key string) error {
	_, err := s.db.Exec(`UPDATE audit_anchors SET s3_key = ? WHERE id = ?`, key, id)
	return err
}

// A restore replaces app.db, which holds these settings. Like the S3 backup
// settings, the running instance's anchoring settings are kept instead of the
// archive's, so a restore cannot quietly switch anchoring off or re-pin the TSA.
// The receipts table is not carried: receipts made after the archive stay in
// S3, and the integrity scan explains them with the backup_restored event.
const stagedSettingsFile = "carry_audit_anchoring_settings.json"

type carriedSettings struct {
	Present bool         `json:"present"`
	Row     *settingsRow `json:"row,omitempty"`
}

// WriteCarryOver saves the current settings row next to a staged restore. It
// is a backup.StageHook.
func WriteCarryOver(db *sql.DB, stageDir string) error {
	r, err := loadRow(db)
	if err != nil {
		return fmt.Errorf("read audit anchoring settings: %w", err)
	}
	b, err := json.Marshal(carriedSettings{Present: r != nil, Row: r})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stageDir, stagedSettingsFile), b, 0600)
}

// ApplyCarriedSettings writes settings kept across a restore into db and
// removes the carry-over file. It does nothing when no restore was applied.
func ApplyCarriedSettings(db *sql.DB, dataDir string) error {
	carriedSettingsFile := backup.CarriedFileName(stagedSettingsFile)
	path := filepath.Join(dataDir, carriedSettingsFile)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var c carriedSettings
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("parse %s: %w", carriedSettingsFile, err)
	}
	if c.Present && c.Row != nil {
		err = writeRow(db, *c.Row)
	} else {
		_, err = db.Exec(`DELETE FROM audit_anchoring_settings WHERE id = 1`)
	}
	if err != nil {
		return fmt.Errorf("restore audit anchoring settings: %w", err)
	}
	if err := os.Remove(path); err != nil {
		log.Printf("WARNING: audit anchoring settings restored, but %s could not be removed: %v", carriedSettingsFile, err)
	}
	return nil
}
