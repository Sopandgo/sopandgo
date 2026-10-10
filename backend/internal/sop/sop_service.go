package sop

import (
	"fmt"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/google/uuid"
)

func (s *Service) RegisterSOP(
	title string,
	actorUserID *string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// Create Sop Record
	if err := createSOPRecord(tx, id, title, createdAt); err != nil {
		return "", fmt.Errorf("failed to create sop: %w", err)
	}

	// Audit Log
	err = s.auditLogger.Log(
		tx,
		audit.EventSOPCreated,
		audit.EntitySOP,
		id,
		actorUserID,
		map[string]string{
			"title": title,
		},
	)
	if err != nil {
		return "", fmt.Errorf("audit log failed: %w", err)
	}

	return id, tx.Commit()
}

func (s *Service) GetSOPByID(id string) (*SOP, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPByIDRecord(s.db, id)
}

func (s *Service) ListSOPs(
	actorUserID string,
	limit, offset int,
	tagID string,
	searchQuery string,
	favoritesOnly, favoritesFirst bool,
) ([]SOPListItem, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return listSOPsRecord(s.db, actorUserID, limit, offset, tagID, searchQuery, favoritesOnly, favoritesFirst)
}

func (s *Service) GetSOPByIDWithTags(id string, viewerUserID string) (*SOPWithTags, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	base, err := getSOPByIDRecord(s.db, id) // returns your existing *SOP (minimal)
	if err != nil {
		return nil, err
	}

	tags, err := getTagsForSOPRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	isFav, err := isSOPFavoritedRecord(s.db, viewerUserID, id)
	if err != nil {
		return nil, err
	}

	latest, published, err := getSOPVersionPointersRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	out := &SOPWithTags{
		ID:            base.ID,
		Title:         base.Title,
		CreatedAt:     base.CreatedAt,
		Tags:          tags,
		IsFavorite:    isFav,
		LatestVersion: latest,
	}
	if published != nil {
		out.PublishedVersion = &published.Version
		out.PublishedVersionID = &published.ID
	}

	return out, nil
}
