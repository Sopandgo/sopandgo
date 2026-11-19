package backup

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// A restore replaces the whole audit chain with the archive's. Anchors taken
// after the archive was made then point at rows that no longer exist or now
// differ, which looks exactly like tampering. To keep that explainable, the
// startup restore reads the live chain tip just before overwriting app.db, and
// RecordRestoreInAudit writes a backup_restored event into the restored chain
// with both tips. The integrity scan uses it to report anchors replaced by a
// restore as an explained break for a person to review, not as a silent pass.
const restoreAuditTipFile = "_restore_audit_tip.json"

// RestoreRecord is the backup_restored payload.
type RestoreRecord struct {
	OldTipID         int64  `json:"old_tip_id"`
	OldTipHash       string `json:"old_tip_hash"`
	ArchiveTipID     int64  `json:"archive_tip_id"`
	ArchiveTipHash   string `json:"archive_tip_hash"`
	ArchiveCreatedAt string `json:"archive_created_at,omitempty"`
	AppliedAt        string `json:"applied_at"`
}

// StageHook runs while a restore is staged, with the live database and the
// staging directory. Files it writes named carry_*.json are moved into the
// data directory as _restore_carry_*.json when the restore is applied, for the
// package that wrote them to pick up after migrations.
type StageHook func(db *sql.DB, pendingDir string) error

// AddStageHook registers h to run on every StageImportApply.
func (s *Service) AddStageHook(h StageHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stageHooks = append(s.stageHooks, h)
}

// CarriedFileName is the data-directory name a staged carry_*.json file is
// moved to when the restore is applied.
func CarriedFileName(staged string) string { return "_restore_" + staged }

func moveCarryFiles(pendingDir, dataDir string) error {
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "carry_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if err := copyFile(filepath.Join(pendingDir, name), filepath.Join(dataDir, CarriedFileName(name))); err != nil {
			return fmt.Errorf("keep %s across restore: %w", name, err)
		}
	}
	return nil
}

// writeRestoreAuditTip reads the live chain tip from the database about to be
// replaced. A missing or unreadable database records an empty tip rather than
// blocking the restore.
func writeRestoreAuditTip(dataDir, pendingDir string) error {
	rec := RestoreRecord{AppliedAt: time.Now().UTC().Format(time.RFC3339Nano), OldTipHash: audit.GenesisHash}

	if b, err := os.ReadFile(filepath.Join(pendingDir, "manifest.json")); err == nil {
		var m Manifest
		if json.Unmarshal(b, &m) == nil {
			rec.ArchiveCreatedAt = m.CreatedAtUTC
		}
	}

	livePath := filepath.Join(dataDir, "app.db")
	if _, err := os.Stat(livePath); err == nil {
		db, err := sql.Open("sqlite", livePath)
		if err == nil {
			err = db.QueryRow(`SELECT id, hash FROM audit_events ORDER BY id DESC LIMIT 1`).Scan(&rec.OldTipID, &rec.OldTipHash)
			_ = db.Close()
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("WARNING: could not read the audit chain tip before restoring: %v", err)
		}
	}

	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, restoreAuditTipFile), b, 0600)
}

// RecordRestoreInAudit writes backup_restored into the restored chain when a
// restore was applied at this start, and removes the record file. It returns
// nil when no restore was applied.
func RecordRestoreInAudit(dataDir string, logger *audit.Logger) (*RestoreRecord, error) {
	path := filepath.Join(dataDir, restoreAuditTipFile)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec RestoreRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, fmt.Errorf("parse %s: %w", restoreAuditTipFile, err)
	}
	rec.ArchiveTipID, rec.ArchiveTipHash, err = logger.Tip()
	if err != nil {
		return nil, err
	}
	if err := logger.Log(nil, audit.EventBackupRestored, audit.EntitySystem, "backup", nil, rec); err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		log.Printf("WARNING: restore recorded in the audit log, but %s could not be removed: %v", restoreAuditTipFile, err)
	}
	return &rec, nil
}
