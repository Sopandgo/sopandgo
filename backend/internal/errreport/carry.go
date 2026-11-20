package errreport

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sopandgo/sopandgo/backend/internal/backup"
)

// A restore replaces app.db, which holds these settings. Like the S3 backup
// and audit anchoring settings, the running instance's are kept instead of
// the archive's: the DSN names this instance's project, and restoring another
// lab's backup must not send this instance's errors there.
const stagedSettingsFile = "carry_error_reporting_settings.json"

type carriedSettings struct {
	Present bool      `json:"present"`
	Row     *Settings `json:"row,omitempty"`
}

// WriteCarryOver saves the current settings row next to a staged restore. It
// is a backup.StageHook.
func WriteCarryOver(db *sql.DB, stageDir string) error {
	r, err := loadRow(db)
	if err != nil {
		return fmt.Errorf("read error reporting settings: %w", err)
	}
	b, err := json.Marshal(carriedSettings{Present: r != nil, Row: r})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stageDir, stagedSettingsFile), b, 0600)
}

// ApplyCarriedSettings writes settings kept across a restore into db and
// removes the carry-over file. It does nothing when no restore was applied.
// Call it at startup, before NewService.
func ApplyCarriedSettings(db *sql.DB, dataDir string) error {
	name := backup.CarriedFileName(stagedSettingsFile)
	path := filepath.Join(dataDir, name)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var c carriedSettings
	if err := json.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	if c.Present && c.Row != nil {
		err = writeRow(db, *c.Row)
	} else {
		_, err = db.Exec(`DELETE FROM error_reporting_settings WHERE id = 1`)
	}
	if err != nil {
		return fmt.Errorf("restore error reporting settings: %w", err)
	}
	if err := os.Remove(path); err != nil {
		log.Printf("WARNING: error reporting settings restored, but %s could not be removed: %v", name, err)
	}
	return nil
}
