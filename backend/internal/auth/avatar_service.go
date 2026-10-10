package auth

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// SetAvatar processes an uploaded image into four JPEG sizes, writes them under
// DATA_DIR/users/<userID>/, and updates avatar_path / avatar_content_hash.
func (s *Service) SetAvatar(userID string, content []byte, actorUserID *string) error {
	sizes, err := processAvatarSizes(content)
	if err != nil {
		return err
	}

	relDir := filepath.ToSlash(filepath.Join("users", userID))
	absDir, err := storage.SafeJoin(s.dataDir, relDir)
	if err != nil {
		return fmt.Errorf("invalid avatar path: %w", err)
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return fmt.Errorf("failed to create avatar directory: %w", err)
	}

	// Write to .tmp first so a failed DB update does not destroy an existing avatar.
	tmpPaths := make([]string, 0, len(avatarPixelSizes))
	finalPaths := make([]string, 0, len(avatarPixelSizes))
	cleanupTmp := func() {
		for _, p := range tmpPaths {
			_ = os.Remove(p)
		}
	}

	var hash1024 string
	for _, px := range avatarPixelSizes {
		name := avatarFileName(px)
		finalPath := filepath.Join(absDir, name)
		tmpPath := finalPath + ".tmp"
		if err := os.WriteFile(tmpPath, sizes[px], 0644); err != nil {
			cleanupTmp()
			return fmt.Errorf("failed to write avatar: %w", err)
		}
		tmpPaths = append(tmpPaths, tmpPath)
		finalPaths = append(finalPaths, finalPath)
		if px == 1024 {
			hash1024 = integrity.HashBytes(sizes[px])
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		cleanupTmp()
		return err
	}
	defer tx.Rollback()

	if _, err := getUserByIDRecord(tx, userID); err != nil {
		cleanupTmp()
		return err
	}

	if err := setUserAvatarRecord(tx, userID, relDir, hash1024); err != nil {
		cleanupTmp()
		return fmt.Errorf("failed to update avatar metadata: %w", err)
	}

	actor := actorUserID
	if actor == nil {
		actor = &userID
	}
	if err := s.auditLogger.Log(
		tx,
		audit.EventUserAvatarUpdated,
		audit.EntityUser,
		userID,
		actor,
		map[string]string{
			"content_hash": hash1024,
			"path":         relDir,
		},
	); err != nil {
		cleanupTmp()
		return err
	}

	if err := tx.Commit(); err != nil {
		cleanupTmp()
		return err
	}

	for i, tmp := range tmpPaths {
		_ = os.Remove(finalPaths[i]) // Windows Rename does not replace existing files
		if err := os.Rename(tmp, finalPaths[i]); err != nil {
			cleanupTmp()
			return fmt.Errorf("failed to finalize avatar file: %w", err)
		}
	}
	return nil
}

// RemoveAvatar clears avatar metadata and deletes size files from disk.
func (s *Service) RemoveAvatar(userID string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	path, _, err := getUserAvatarMetaRecord(tx, userID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found: %s", userID)
	}
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("no avatar")
	}

	if err := clearUserAvatarRecord(tx, userID); err != nil {
		return err
	}

	actor := actorUserID
	if actor == nil {
		actor = &userID
	}
	if err := s.auditLogger.Log(
		tx,
		audit.EventUserAvatarRemoved,
		audit.EntityUser,
		userID,
		actor,
		map[string]string{"path": path},
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	absDir, err := storage.SafeJoin(s.dataDir, path)
	if err == nil {
		for _, px := range avatarPixelSizes {
			_ = os.Remove(filepath.Join(absDir, avatarFileName(px)))
		}
		_ = os.Remove(absDir) // remove dir if empty; ignore errors
	}
	return nil
}

// AvatarAbsolutePath returns the absolute path for a size (sm|md|lg|xl) and the
// content hash used for ETag. Returns sql.ErrNoRows-style "no avatar" when unset.
func (s *Service) AvatarAbsolutePath(userID, sizeName string) (absPath, contentHash string, err error) {
	px, ok := AvatarSizeName[sizeName]
	if !ok {
		return "", "", fmt.Errorf("invalid size")
	}

	path, hash, err := getUserAvatarMetaRecord(s.db, userID)
	if err != nil {
		return "", "", err
	}
	if path == "" {
		return "", "", fmt.Errorf("no avatar")
	}

	relFile := filepath.ToSlash(filepath.Join(path, avatarFileName(px)))
	absPath, err = storage.SafeJoin(s.dataDir, relFile)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", "", err
	}
	return absPath, hash, nil
}
