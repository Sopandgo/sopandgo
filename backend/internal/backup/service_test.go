package backup

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func newTestService(t *testing.T) (*Service, *storage.Storage, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.DB.Close() })
	svc := NewService(store.DB, dir, "testenv", storage.LatestSchemaVersion())
	return svc, store, dir
}

func copyZipToFile(t *testing.T, srcZip, dst string) {
	t.Helper()
	in, err := os.Open(srcZip)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeBackupZip(t *testing.T, zipPath, dbFile string, manifest Manifest) {
	t.Helper()
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if err := addFileToZip(zw, dbFile, "app.db"); err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	if _, err := mw.Write(b); err != nil {
		_ = zw.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateImportCompatibility(t *testing.T) {
	svc := NewService(nil, "", "x", 10)
	cases := []struct {
		name    string
		m       Manifest
		maxSch  int
		wantErr error
	}{
		{
			name: "ok",
			m: Manifest{
				BackupFormatVersion: FormatVersion,
				DBSchemaVersion:     5,
			},
			maxSch:  10,
			wantErr: nil,
		},
		{
			name: "bad format",
			m: Manifest{
				BackupFormatVersion: 99,
				DBSchemaVersion:     1,
			},
			maxSch:  10,
			wantErr: ErrUnsupportedBackupFormat,
		},
		{
			name: "schema too new",
			m: Manifest{
				BackupFormatVersion: FormatVersion,
				DBSchemaVersion:     50,
			},
			maxSch:  10,
			wantErr: ErrSchemaTooNew,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc.maxSupportedSchema = tc.maxSch
			err := svc.validateImportCompatibility(tc.m)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestTryLock_BlocksExportZip(t *testing.T) {
	svc, _, _ := newTestService(t)
	if !svc.TryLock("hold") {
		t.Fatal("expected TryLock to succeed")
	}
	_, _, _, err := svc.ExportZip()
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("got %v, want ErrBusy", err)
	}
	svc.Unlock()
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Dir(zipPath))
}

func TestExportZip_ContainsDBAndManifest(t *testing.T) {
	svc, _, dir := newTestService(t)
	sopsDir := filepath.Join(dir, "sops")
	if err := os.MkdirAll(sopsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sopsDir, "note.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	zipPath, fileName, manifest, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	if manifest.BackupFormatVersion != FormatVersion {
		t.Fatalf("format %d", manifest.BackupFormatVersion)
	}
	if manifest.DBSchemaVersion != storage.LatestSchemaVersion() {
		t.Fatalf("schema %d", manifest.DBSchemaVersion)
	}
	if manifest.AppVersion != "testenv" {
		t.Fatalf("app version %q", manifest.AppVersion)
	}
	if fileName == "" || filepath.Ext(fileName) != ".zip" {
		t.Fatalf("fileName %q", fileName)
	}

	mf, err := svc.readManifestFromZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if mf.DBSchemaVersion != manifest.DBSchemaVersion {
		t.Fatal("manifest mismatch")
	}
}

func TestValidateImportArchive_RoundTrip(t *testing.T) {
	svc, _, _ := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	f, err := os.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	res, err := svc.ValidateImportArchive(f)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.BackupFormatVersion != FormatVersion {
		t.Fatal(res.Manifest)
	}
}

func TestValidateImportArchive_ErrPendingRestore(t *testing.T) {
	svc, _, dir := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(dir, "keep.zip")
	copyZipToFile(t, zipPath, saved)
	_ = os.RemoveAll(filepath.Dir(zipPath))

	if err := os.MkdirAll(filepath.Join(dir, "_restore_pending"), 0755); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = svc.ValidateImportArchive(f)
	if !errors.Is(err, ErrPendingRestore) {
		t.Fatalf("got %v, want ErrPendingRestore", err)
	}
}

func TestValidateImportArchive_ErrUnsupportedFormat(t *testing.T) {
	svc, _, dir := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	tmp := filepath.Join(dir, "mod.zip")
	if err := alterManifestInZip(t, zipPath, tmp, func(m *Manifest) {
		m.BackupFormatVersion = 999
	}); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = svc.ValidateImportArchive(f)
	if !errors.Is(err, ErrUnsupportedBackupFormat) {
		t.Fatalf("got %v, want ErrUnsupportedBackupFormat", err)
	}
}

func TestValidateImportArchive_ErrSchemaTooNew(t *testing.T) {
	svc, _, dir := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	tmp := filepath.Join(dir, "mod.zip")
	if err := alterManifestInZip(t, zipPath, tmp, func(m *Manifest) {
		m.DBSchemaVersion = storage.LatestSchemaVersion() + 100
	}); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = svc.ValidateImportArchive(f)
	if !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("got %v, want ErrSchemaTooNew", err)
	}
}

// alterManifestInZip reads an existing backup zip, replaces manifest.json, writes to dstPath.
func alterManifestInZip(t *testing.T, srcPath, dstPath string, fn func(*Manifest)) error {
	t.Helper()
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return err
	}
	defer r.Close()

	var manifest Manifest
	var dbData []byte
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
		switch name {
		case "manifest.json":
			if err := json.Unmarshal(b, &manifest); err != nil {
				return err
			}
		case "app.db":
			dbData = b
		}
	}
	fn(&manifest)
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	if len(dbData) > 0 {
		w, err := zw.Create("app.db")
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			return err
		}
		if _, err := w.Write(dbData); err != nil {
			_ = zw.Close()
			_ = out.Close()
			return err
		}
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		_ = zw.Close()
		_ = out.Close()
		return err
	}
	if _, err := mw.Write(b); err != nil {
		_ = zw.Close()
		_ = out.Close()
		return err
	}
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if name == "app.db" || name == "manifest.json" {
			continue
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			return err
		}
		rc, err := f.Open()
		if err != nil {
			_ = zw.Close()
			_ = out.Close()
			return err
		}
		if _, err := io.Copy(w, rc); err != nil {
			_ = rc.Close()
			_ = zw.Close()
			_ = out.Close()
			return err
		}
		_ = rc.Close()
	}
	if err := zw.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func TestStageImportApply_WrongConfirmation(t *testing.T) {
	svc, _, _ := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	f, err := os.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = svc.StageImportApply(f, "wrong")
	if !errors.Is(err, ErrApplyConfirmation) {
		t.Fatalf("got %v, want ErrApplyConfirmation", err)
	}
}

func TestStageImportApply_CreatesPendingAndSnapshot(t *testing.T) {
	svc, _, dir := newTestService(t)
	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(zipPath)) }()

	f, err := os.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.StageImportApply(f, "APPLY BACKUP")
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !res.RequiresRestart {
		t.Fatal("expected RequiresRestart")
	}
	if res.PreApplyBackup == "" {
		t.Fatal("expected PreApplyBackup name")
	}

	pending := filepath.Join(dir, "_restore_pending")
	st, err := os.Stat(pending)
	if err != nil || !st.IsDir() {
		t.Fatalf("pending dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pending, "app.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pending, "manifest.json")); err != nil {
		t.Fatal(err)
	}

	snaps, err := os.ReadDir(filepath.Join(dir, "_backup_snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range snaps {
		if filepath.Ext(e.Name()) == ".zip" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected pre-apply snapshot zip")
	}
}

func TestStageImportApply_ErrInvalidArchiveEmptyDB(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	svc := NewService(store.DB, dir, "t", storage.LatestSchemaVersion())

	emptyDB := filepath.Join(dir, "empty.db")
	if err := os.WriteFile(emptyDB, nil, 0644); err != nil {
		t.Fatal(err)
	}
	zpath := filepath.Join(dir, "bad.zip")
	writeBackupZip(t, zpath, emptyDB, Manifest{
		BackupFormatVersion: FormatVersion,
		DBSchemaVersion:     storage.LatestSchemaVersion(),
		AppVersion:          "x",
		CreatedAtUTC:        "2000-01-01T00:00:00Z",
		DataLayoutVersion:   1,
	})

	f, err := os.Open(zpath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	_, err = svc.StageImportApply(f, "APPLY BACKUP")
	if !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestApplyPendingRestoreAtStartup_RestoresDB(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	db := store.DB
	svc := NewService(db, dir, "t", storage.LatestSchemaVersion())

	zipPath, _, _, err := svc.ExportZip()
	if err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(dir, "snapshot.zip")
	copyZipToFile(t, zipPath, saved)
	_ = os.RemoveAll(filepath.Dir(zipPath))

	if _, err := db.Exec(`UPDATE app_settings SET mail_mode = 'manual_links', updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var mode string
	if err := db.QueryRow(`SELECT mail_mode FROM app_settings WHERE id = 1`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "manual_links" {
		t.Fatalf("after update got %q", mode)
	}

	f, err := os.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.StageImportApply(f, "APPLY BACKUP")
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRow(`SELECT mail_mode FROM app_settings WHERE id = 1`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "manual_links" {
		t.Fatalf("live db should stay manual_links until restart apply, got %q", mode)
	}

	_ = db.Close()

	if err := os.WriteFile(filepath.Join(dir, "app.db-wal"), []byte("wal"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.db-shm"), []byte("shm"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyPendingRestoreAtStartup(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_restore_pending")); !os.IsNotExist(err) {
		t.Fatal("expected pending dir removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "app.db-wal")); !os.IsNotExist(err) {
		t.Fatal("expected wal removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "app.db-shm")); !os.IsNotExist(err) {
		t.Fatal("expected shm removed")
	}

	store2, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.DB.Close()
	if err := store2.DB.QueryRow(`SELECT mail_mode FROM app_settings WHERE id = 1`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "smtp" {
		t.Fatalf("after apply want smtp, got %q", mode)
	}
}

func TestApplyPendingRestoreAtStartup_NoOpWithoutPending(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()

	if err := ApplyPendingRestoreAtStartup(dir); err != nil {
		t.Fatal(err)
	}
}

func TestExportZip_ConcurrentOneBusy(t *testing.T) {
	svc, _, _ := newTestService(t)
	var ok, busy int32
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			zipPath, _, _, err := svc.ExportZip()
			if zipPath != "" {
				_ = os.RemoveAll(filepath.Dir(zipPath))
			}
			if errors.Is(err, ErrBusy) {
				atomic.AddInt32(&busy, 1)
				return
			}
			if err == nil {
				atomic.AddInt32(&ok, 1)
				return
			}
			t.Errorf("unexpected error: %v", err)
		}()
	}
	wg.Wait()
	if ok != 1 || busy != 1 {
		t.Fatalf("ok=%d busy=%d", ok, busy)
	}
}
