package backup

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadManifestFromZip_MissingAppDB(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "x.zip")

	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewEncoder(w).Encode(Manifest{
		BackupFormatVersion: FormatVersion,
		DBSchemaVersion:     1,
	})
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	svc := &Service{}
	_, err = svc.readManifestFromZip(zpath)
	if !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestReadManifestFromZip_MissingManifest(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	if err := os.WriteFile(dbPath, []byte("not-sqlite"), 0644); err != nil {
		t.Fatal(err)
	}
	zpath := filepath.Join(dir, "x.zip")

	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if err := addFileToZip(zw, dbPath, "app.db"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	svc := &Service{}
	_, err = svc.readManifestFromZip(zpath)
	if !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestReadManifestFromZip_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	if err := os.WriteFile(dbPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	zpath := filepath.Join(dir, "x.zip")

	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if err := addFileToZip(zw, dbPath, "app.db"); err != nil {
		t.Fatal(err)
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = mw.Write([]byte(`{`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	svc := &Service{}
	_, err = svc.readManifestFromZip(zpath)
	if !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestExtractZip_RejectsParentPath(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "evil.zip")
	dst := filepath.Join(dir, "out")

	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../outside.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("nope"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	err = extractZip(zpath, dst)
	if !errors.Is(err, ErrInvalidBackupArchive) {
		t.Fatalf("got %v, want ErrInvalidBackupArchive", err)
	}
}

func TestExtractZip_WritesNestedFile(t *testing.T) {
	dir := t.TempDir()
	zpath := filepath.Join(dir, "ok.zip")
	dst := filepath.Join(dir, "out")

	f, err := os.Create(zpath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("sops/sub/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("hello"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := extractZip(zpath, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "sops/sub/hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("content %q", got)
	}
}
