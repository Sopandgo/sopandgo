package sop

import (
	"fmt"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// FavoriteSOP adds the SOP to the user's favorites (idempotent).
func (s *Service) FavoriteSOP(sopID, actorUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := getSOPByIDRecord(tx, sopID); err != nil {
		return err
	}

	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	n, err := addSOPFavoriteRecord(tx, actorUserID, sopID, createdAt)
	if err != nil {
		return fmt.Errorf("failed to favorite sop: %w", err)
	}

	if n > 0 {
		if err := s.auditLogger.Log(
			tx,
			audit.EventSOPFavoriteAdded,
			audit.EntitySOP,
			sopID,
			&actorUserID,
			nil,
		); err != nil {
			return fmt.Errorf("audit log failed: %w", err)
		}
	}

	return tx.Commit()
}

// UnfavoriteSOP removes the SOP from the user's favorites (idempotent).
func (s *Service) UnfavoriteSOP(sopID, actorUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	n, err := removeSOPFavoriteRecord(tx, actorUserID, sopID)
	if err != nil {
		return fmt.Errorf("failed to unfavorite sop: %w", err)
	}

	if n > 0 {
		if err := s.auditLogger.Log(
			tx,
			audit.EventSOPFavoriteRemoved,
			audit.EntitySOP,
			sopID,
			&actorUserID,
			nil,
		); err != nil {
			return fmt.Errorf("audit log failed: %w", err)
		}
	}

	return tx.Commit()
}
