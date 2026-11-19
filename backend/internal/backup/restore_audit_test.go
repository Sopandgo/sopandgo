package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// A restore replaces the audit chain. The live tip read just before the swap
// and the restored tip both end up in a backup_restored event, and files a
// stage hook wrote are carried into the data directory.
func TestRestore_RecordsBothChainTipsAndCarriesHookFiles(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	logger, err := audit.New(store.DB)
	if err != nil {
		t.Fatal(err)
	}
	logN := func(l *audit.Logger, n int) {
		for i := 0; i < n; i++ {
			if err := l.Log(nil, "login", "user", "u", nil, "{}"); err != nil {
				t.Fatal(err)
			}
		}
	}

	svc := NewService(store.DB, dir, "t", storage.LatestSchemaVersion())
	svc.AddStageHook(func(db *sql.DB, pendingDir string) error {
		return os.WriteFile(filepath.Join(pendingDir, "carry_test.json"), []byte(`{"kept":true}`), 0600)
	})

	logN(logger, 3)
	archiveTipID, archiveTipHash, _ := logger.Tip()
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "snapshot.zip")
	copyZipToFile(t, zipPath, archive)
	_ = os.RemoveAll(filepath.Dir(zipPath))

	logN(logger, 4) // history the restore will throw away
	f, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StageImportApply(f, "RESTORE BACKUP"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	logN(logger, 1) // e.g. backup_apply_staged, written after staging
	oldTipID, oldTipHash, _ := logger.Tip()
	_ = store.DB.Close()

	if err := ApplyPendingRestoreAtStartup(dir); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, CarriedFileName("carry_test.json"))); err != nil || string(b) != `{"kept":true}` {
		t.Fatalf("carried hook file = %q, %v", b, err)
	}

	store2, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.DB.Close()
	logger2, err := audit.New(store2.DB)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := RecordRestoreInAudit(dir, logger2)
	if err != nil || rec == nil {
		t.Fatalf("RecordRestoreInAudit = %+v, %v", rec, err)
	}
	if rec.OldTipID != oldTipID || rec.OldTipHash != oldTipHash {
		t.Errorf("old tip = %d %s, want %d %s", rec.OldTipID, rec.OldTipHash, oldTipID, oldTipHash)
	}
	if rec.ArchiveTipID != archiveTipID || rec.ArchiveTipHash != archiveTipHash {
		t.Errorf("archive tip = %d %s, want %d %s", rec.ArchiveTipID, rec.ArchiveTipHash, archiveTipID, archiveTipHash)
	}
	if rec.ArchiveCreatedAt == "" {
		t.Error("archive created_at missing")
	}

	var n int
	store2.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'backup_restored'`).Scan(&n)
	if n != 1 {
		t.Fatalf("backup_restored events = %d", n)
	}
	if report, _ := logger2.VerifyEntireChain(); report.Invalid != 0 {
		t.Fatalf("restored chain = %+v", report)
	}

	// Recorded once only.
	if rec, err := RecordRestoreInAudit(dir, logger2); err != nil || rec != nil {
		t.Fatalf("second call = %+v, %v", rec, err)
	}
}
