package errreport_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/errreport"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// openDB opens a database of its own. OpenInMemory would not do: every
// in-memory store shares one database.
func openDB(t *testing.T) *sql.DB {
	t.Helper()
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.DB.Close() })
	return store.DB
}

func saveSettings(t *testing.T, db *sql.DB, in errreport.SaveInput) {
	t.Helper()
	svc, err := errreport.NewService(db, errreport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if _, err := svc.Save(in); err != nil {
		t.Fatal(err)
	}
}

func getSettings(t *testing.T, db *sql.DB) errreport.Settings {
	t.Helper()
	svc, err := errreport.NewService(db, errreport.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	st, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// restore stages a restore from the running db, then applies it to the
// archive's db the way the backup service and startup do.
func restore(t *testing.T, running, archive *sql.DB) string {
	t.Helper()
	stageDir, dataDir := t.TempDir(), t.TempDir()
	if err := errreport.WriteCarryOver(running, stageDir); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(stageDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "carry_") {
			b, _ := os.ReadFile(filepath.Join(stageDir, e.Name()))
			if err := os.WriteFile(filepath.Join(dataDir, backup.CarriedFileName(e.Name())), b, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := errreport.ApplyCarriedSettings(archive, dataDir); err != nil {
		t.Fatal(err)
	}
	return dataDir
}

func TestRestoreKeepsThisInstancesSettings(t *testing.T) {
	running, archive := openDB(t), openDB(t)
	saveSettings(t, running, errreport.SaveInput{Enabled: true, DSN: testDSN, Environment: "lab-a"})
	saveSettings(t, archive, errreport.SaveInput{Enabled: true, DSN: "https://other@sentry.example.com/9", Environment: "other-lab"})

	dataDir := restore(t, running, archive)

	if got := getSettings(t, archive); got.DSN != testDSN || got.Environment != "lab-a" || !got.Enabled {
		t.Fatalf("after restore = %+v, want this instance's settings", got)
	}
	if left, _ := os.ReadDir(dataDir); len(left) != 0 {
		t.Fatalf("carry-over file left behind: %v", left)
	}
}

func TestRestoreOnAnInstanceWithoutSettingsTurnsTheArchivesOff(t *testing.T) {
	running, archive := openDB(t), openDB(t)
	saveSettings(t, archive, errreport.SaveInput{Enabled: true, DSN: "https://other@sentry.example.com/9", Environment: "other-lab"})

	restore(t, running, archive)

	if got := getSettings(t, archive); got.Enabled || got.DSN != "" {
		t.Fatalf("after restore = %+v, want the defaults", got)
	}
}

func TestApplyCarriedSettingsIsANoOpWithoutARestore(t *testing.T) {
	db := openDB(t)
	saveSettings(t, db, errreport.SaveInput{Enabled: true, DSN: testDSN, Environment: "lab-a"})
	if err := errreport.ApplyCarriedSettings(db, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if got := getSettings(t, db); got.DSN != testDSN {
		t.Fatalf("settings changed: %+v", got)
	}
}
