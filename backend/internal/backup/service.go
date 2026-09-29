package backup

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	FormatVersion = 1
)

var (
	ErrBusy                    = errors.New("backup operation already running")
	ErrInvalidBackupArchive    = errors.New("invalid backup archive")
	ErrUnsupportedBackupFormat = errors.New("unsupported backup format version")
	ErrSchemaTooNew            = errors.New("backup schema is newer than this app supports")
	ErrApplyConfirmation       = errors.New(`confirmation must equal "APPLY BACKUP"`)
	ErrPendingRestore          = errors.New("a backup restore is already staged and waiting for restart")
)

type Manifest struct {
	BackupFormatVersion int    `json:"backup_format_version"`
	AppVersion          string `json:"app_version"`
	CreatedAtUTC        string `json:"created_at_utc"`
	DBSchemaVersion     int    `json:"db_schema_version"`
	DataLayoutVersion   int    `json:"data_layout_version"`
}

type ImportResult struct {
	Manifest Manifest `json:"manifest"`
}

type ApplyResult struct {
	Manifest        Manifest `json:"manifest"`
	RequiresRestart bool     `json:"requires_restart"`
	PreApplyBackup  string   `json:"pre_apply_backup"`
}

type Service struct {
	db                 *sql.DB
	dataDir            string
	appVersion         string
	maxSupportedSchema int

	mu      sync.Mutex
	locked  bool
	message string
}

func NewService(db *sql.DB, dataDir, appVersion string, maxSupportedSchema int) *Service {
	return &Service{
		db:                 db,
		dataDir:            dataDir,
		appVersion:         strings.TrimSpace(appVersion),
		maxSupportedSchema: maxSupportedSchema,
	}
}

func (s *Service) IsLocked() (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locked, s.message
}

func (s *Service) HasPendingRestore() bool {
	pendingDir := filepath.Join(s.dataDir, "_restore_pending")
	st, err := os.Stat(pendingDir)
	return err == nil && st.IsDir()
}

func (s *Service) TryLock(reason string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.locked {
		return false
	}
	s.locked = true
	s.message = reason
	return true
}

func (s *Service) Unlock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locked = false
	s.message = ""
}

func (s *Service) ExportZip() (zipPath string, fileName string, manifest Manifest, err error) {
	if !s.TryLock("backup export in progress") {
		return "", "", Manifest{}, ErrBusy
	}
	defer s.Unlock()

	return s.exportZipUnlocked()
}

func (s *Service) ValidateImportArchive(file multipart.File) (ImportResult, error) {
	if s.HasPendingRestore() {
		return ImportResult{}, ErrPendingRestore
	}
	if !s.TryLock("backup import validation in progress") {
		return ImportResult{}, ErrBusy
	}
	defer s.Unlock()

	tmpDir, err := os.MkdirTemp("", "sopandgo-import-")
	if err != nil {
		return ImportResult{}, err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "upload.zip")
	out, err := os.Create(zipPath)
	if err != nil {
		return ImportResult{}, err
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		return ImportResult{}, err
	}
	if err := out.Close(); err != nil {
		return ImportResult{}, err
	}

	manifest, err := s.readManifestFromZip(zipPath)
	if err != nil {
		return ImportResult{}, err
	}
	if err := s.validateImportCompatibility(manifest); err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Manifest: manifest}, nil
}

func (s *Service) StageImportApply(file multipart.File, confirmation string) (ApplyResult, error) {
	if strings.TrimSpace(confirmation) != "APPLY BACKUP" {
		return ApplyResult{}, ErrApplyConfirmation
	}
	if s.HasPendingRestore() {
		return ApplyResult{}, ErrPendingRestore
	}
	if !s.TryLock("backup apply staging in progress") {
		return ApplyResult{}, ErrBusy
	}
	defer s.Unlock()

	tmpDir, err := os.MkdirTemp("", "sopandgo-import-apply-")
	if err != nil {
		return ApplyResult{}, err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "upload.zip")
	if err := saveMultipartToPath(file, zipPath); err != nil {
		return ApplyResult{}, err
	}

	manifest, err := s.readManifestFromZip(zipPath)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := s.validateImportCompatibility(manifest); err != nil {
		return ApplyResult{}, err
	}

	preApplyPath, err := s.createPersistentSnapshotUnlocked("pre-apply")
	if err != nil {
		return ApplyResult{}, err
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := extractZip(zipPath, extractDir); err != nil {
		return ApplyResult{}, ErrInvalidBackupArchive
	}

	stagedDB := filepath.Join(extractDir, "app.db")
	if _, err := os.Stat(stagedDB); err != nil {
		return ApplyResult{}, ErrInvalidBackupArchive
	}
	if _, err := readSchemaVersionFromDBFile(stagedDB); err != nil {
		return ApplyResult{}, ErrInvalidBackupArchive
	}

	pendingDir := filepath.Join(s.dataDir, "_restore_pending")
	if err := os.RemoveAll(pendingDir); err != nil {
		return ApplyResult{}, err
	}
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return ApplyResult{}, err
	}

	if err := copyFile(stagedDB, filepath.Join(pendingDir, "app.db")); err != nil {
		return ApplyResult{}, err
	}
	stagedSops := filepath.Join(extractDir, "sops")
	if st, err := os.Stat(stagedSops); err == nil && st.IsDir() {
		if err := copyDir(stagedSops, filepath.Join(pendingDir, "sops")); err != nil {
			return ApplyResult{}, err
		}
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return ApplyResult{}, err
	}
	if err := os.WriteFile(filepath.Join(pendingDir, "manifest.json"), manifestBytes, 0644); err != nil {
		return ApplyResult{}, err
	}

	return ApplyResult{
		Manifest:        manifest,
		RequiresRestart: true,
		PreApplyBackup:  filepath.Base(preApplyPath),
	}, nil
}

func ApplyPendingRestoreAtStartup(dataDir string) error {
	pendingDir := filepath.Join(dataDir, "_restore_pending")
	if st, err := os.Stat(pendingDir); err != nil || !st.IsDir() {
		return nil
	}

	srcDB := filepath.Join(pendingDir, "app.db")
	if _, err := os.Stat(srcDB); err != nil {
		return fmt.Errorf("restore pending but app.db missing: %w", err)
	}

	dstDB := filepath.Join(dataDir, "app.db")
	if err := copyFile(srcDB, dstDB); err != nil {
		return fmt.Errorf("failed to apply pending app.db: %w", err)
	}

	_ = os.Remove(filepath.Join(dataDir, "app.db-wal"))
	_ = os.Remove(filepath.Join(dataDir, "app.db-shm"))

	srcSops := filepath.Join(pendingDir, "sops")
	if st, err := os.Stat(srcSops); err == nil && st.IsDir() {
		dstSops := filepath.Join(dataDir, "sops")
		if err := os.RemoveAll(dstSops); err != nil {
			return fmt.Errorf("failed to clear sops directory: %w", err)
		}
		if err := copyDir(srcSops, dstSops); err != nil {
			return fmt.Errorf("failed to restore sops directory: %w", err)
		}
	}

	if err := os.RemoveAll(pendingDir); err != nil {
		log.Printf("WARNING: pending restore applied, but cleanup failed: %v", err)
	}
	return nil
}

func (s *Service) exportZipUnlocked() (zipPath string, fileName string, manifest Manifest, err error) {
	tmpDir, err := os.MkdirTemp("", "sopandgo-export-")
	if err != nil {
		return "", "", Manifest{}, err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	dbSnapPath := filepath.Join(tmpDir, "app.db")
	if err := s.createSQLiteSnapshot(dbSnapPath); err != nil {
		return "", "", Manifest{}, err
	}

	schemaVersion, err := readSchemaVersionFromDBFile(dbSnapPath)
	if err != nil {
		return "", "", Manifest{}, err
	}

	manifest = Manifest{
		BackupFormatVersion: FormatVersion,
		AppVersion:          defaultString(s.appVersion, "dev"),
		CreatedAtUTC:        time.Now().UTC().Format(time.RFC3339),
		DBSchemaVersion:     schemaVersion,
		DataLayoutVersion:   1,
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	fileName = fmt.Sprintf("sopandgo-backup-%s.zip", timestamp)
	zipPath = filepath.Join(tmpDir, fileName)

	if err := s.buildZip(zipPath, dbSnapPath, manifest); err != nil {
		return "", "", Manifest{}, err
	}

	return zipPath, fileName, manifest, nil
}

func (s *Service) createPersistentSnapshotUnlocked(prefix string) (string, error) {
	tmpZipPath, fileName, _, err := s.exportZipUnlocked()
	if err != nil {
		return "", err
	}
	snapshotsDir := filepath.Join(s.dataDir, "_backup_snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(snapshotsDir, prefix+"-"+fileName)
	if err := copyFile(tmpZipPath, target); err != nil {
		return "", err
	}
	return target, nil
}

func (s *Service) buildZip(zipPath, dbSnapPath string, manifest Manifest) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	if err := addFileToZip(zw, dbSnapPath, "app.db"); err != nil {
		return err
	}

	sopsDir := filepath.Join(s.dataDir, "sops")
	if _, err := os.Stat(sopsDir); err == nil {
		if err := addDirToZip(zw, sopsDir, "sops"); err != nil {
			return err
		}
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err := w.Write(manifestBytes); err != nil {
		return err
	}

	return nil
}

func (s *Service) createSQLiteSnapshot(dstPath string) error {
	escaped := strings.ReplaceAll(filepath.ToSlash(dstPath), "'", "''")
	stmt := fmt.Sprintf("VACUUM INTO '%s';", escaped)
	_, err := s.db.Exec(stmt)
	return err
}

func (s *Service) readManifestFromZip(zipPath string) (Manifest, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return Manifest{}, ErrInvalidBackupArchive
	}
	defer zr.Close()

	var (
		hasDB    bool
		manifest Manifest
		found    bool
	)
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if name == "app.db" {
			hasDB = true
		}
		if name == "manifest.json" {
			found = true
			rc, err := f.Open()
			if err != nil {
				return Manifest{}, ErrInvalidBackupArchive
			}
			decErr := json.NewDecoder(rc).Decode(&manifest)
			_ = rc.Close()
			if decErr != nil {
				return Manifest{}, ErrInvalidBackupArchive
			}
		}
	}
	if !hasDB || !found {
		return Manifest{}, ErrInvalidBackupArchive
	}
	return manifest, nil
}

func (s *Service) validateImportCompatibility(manifest Manifest) error {
	if manifest.BackupFormatVersion != FormatVersion {
		return ErrUnsupportedBackupFormat
	}
	if manifest.DBSchemaVersion > s.maxSupportedSchema {
		return ErrSchemaTooNew
	}
	return nil
}

func readSchemaVersionFromDBFile(dbPath string) (int, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var v int
	if err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

func addDirToZip(zw *zip.Writer, sourceDir, zipRoot string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		entryName := filepath.ToSlash(filepath.Join(zipRoot, rel))
		return addFileToZip(zw, path, entryName)
	})
}

func addFileToZip(zw *zip.Writer, sourcePath, entryName string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(entryName)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	f, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func defaultString(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func saveMultipartToPath(file multipart.File, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	return err
}

func extractZip(zipPath, dstDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") {
			return ErrInvalidBackupArchive
		}
		outPath := filepath.Join(dstDir, filepath.FromSlash(name))
		if !strings.HasPrefix(filepath.Clean(outPath), filepath.Clean(dstDir)) {
			return ErrInvalidBackupArchive
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(outPath)
		if err != nil {
			_ = src.Close()
			return err
		}
		_, cpErr := io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		if cpErr != nil {
			return cpErr
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}
