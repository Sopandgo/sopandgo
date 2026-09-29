package sop

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
	"github.com/google/uuid"
)

func (s *Service) AddAsset(
	sopID string,
	filename string,
	content []byte,
	actorUserID *string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// 1. Prepare Paths
	// DB PATH: Relative (Portable) -> "sops/<id>/assets/<filename>"
	relPath := filepath.Join("sops", sopID, "assets", filename)

	dbPath := filepath.ToSlash(relPath)

	// DISK PATH: Absolute (Secure) -> "/data/sops/<id>/assets/<filename>"
	absPath, err := storage.SafeJoin(s.dataDir, relPath)
	if err != nil {
		return "", fmt.Errorf("invalid path construction: %w", err)
	}

	// 2. Prevent overwrite (Check if file exists on disk)
	if _, err := os.Stat(absPath); err == nil {
		return "", fmt.Errorf("asset already exists: %s", filename)
	} else if !os.IsNotExist(err) {
		return "", err
	}

	// 3. Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 4. Write file to disk
	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write asset file: %w", err)
	}

	// CLEANUP: If DB insert fails, delete the file
	commitSuccessful := false
	defer func() {
		if !commitSuccessful {
			os.Remove(absPath)
		}
	}()

	// 5. Compute Hash & Meta
	contentHash := integrity.HashBytes(content)
	assetID := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 6. Insert DB Record
	// Note: We pass 'dbPath' here so the DB stores the portable path
	err = createSOPAssetRecord(
		tx,
		assetID,
		sopID,
		filename,
		dbPath,
		contentHash,
		createdAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create asset record: %w", err)
	}

	// 7. Audit Log
	err = s.auditLogger.Log(
		tx,
		audit.EventAssetAdded,
		audit.EntitySOPAsset,
		assetID,
		actorUserID,
		map[string]string{
			"sop_id":       sopID,
			"path":         relPath,
			"content_hash": contentHash,
		},
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	commitSuccessful = true
	return assetID, nil
}

func (s *Service) GetAssetByID(id string) (*SOPAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPAssetByIDRecord(s.db, id)
}

func (s *Service) GetAssetsBySOPID(sopID string) ([]SOPAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPAssetsBySOPIDRecord(s.db, sopID)
}

// GetAssetContent securely reads the file content from disk
func (s *Service) GetAssetContent(id string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, err := getSOPAssetByIDRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	// Use SafeJoin to ensure the relative path from DB is safe
	absPath, err := storage.SafeJoin(s.dataDir, asset.ContentPath)
	if err != nil {
		return nil, fmt.Errorf("invalid asset path: %w", err)
	}

	return os.ReadFile(absPath)
}

// GetAssetPath resolves the secure absolute path for an asset.
// This allows the handler to use http.ServeFile (better for caching/streaming).
func (s *Service) GetAssetPath(assetID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, err := getSOPAssetByIDRecord(s.db, assetID)
	if err != nil {
		return "", err
	}

	// Securely resolve the path
	return storage.SafeJoin(s.dataDir, asset.ContentPath)
}

// VerifyAssetIntegrity checks if the file on disk matches the DB hash
func (s *Service) VerifyAssetIntegrity(assetID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, err := getSOPAssetByIDRecord(s.db, assetID)
	if err != nil {
		return false, err
	}

	absPath, err := storage.SafeJoin(s.dataDir, asset.ContentPath)
	if err != nil {
		return false, fmt.Errorf("invalid storage path: %w", err)
	}

	return integrity.VerifyFile(absPath, asset.ContentHash)
}
