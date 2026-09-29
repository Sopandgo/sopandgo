package sop

import (
	"fmt"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/google/uuid"
)

// CreateTag creates a new global tag that can be attached to any SOP.
func (s *Service) CreateTag(title string, actorUserID *string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("tag title cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	exists, err := checkTagExistsCaseInsensitiveRecord(tx, title)
	if err != nil {
		return "", fmt.Errorf("failed to check tag uniqueness: %w", err)
	}
	if exists {
		// Return a clear error so the API handler can send a 409 Conflict
		return "", fmt.Errorf("conflict: a tag with this name already exists")
	}

	id := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	if err := createTagRecord(tx, id, title, createdAt); err != nil {
		return "", fmt.Errorf("failed to create tag: %w", err)
	}

	// Log the creation of the tag globally
	err = s.auditLogger.Log(
		tx,
		audit.EventTagCreated,
		audit.EntityTag,
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

// ListTags returns all available global tags.
func (s *Service) ListTags() ([]Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getTagsRecord(s.db)
}

// AttachTagToSOP links an existing tag to an existing SOP.
func (s *Service) AttachTagToSOP(sopID, tagID string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := attachTagToSOPRecord(tx, sopID, tagID); err != nil {
		return fmt.Errorf("failed to attach tag to sop: %w", err)
	}

	// We log this against the SOP entity to track organizational changes
	err = s.auditLogger.Log(
		tx,
		audit.EventTagAttached,
		audit.EntitySOP,
		sopID,
		actorUserID,
		map[string]string{
			"tag_id": tagID,
		},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit()
}

// DetachTagFromSOP removes a tag link from an SOP.
func (s *Service) DetachTagFromSOP(sopID, tagID string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := detachTagFromSOPRecord(tx, sopID, tagID); err != nil {
		return fmt.Errorf("failed to detach tag from sop: %w", err)
	}

	err = s.auditLogger.Log(
		tx,
		audit.EventTagDetached,
		audit.EntitySOP,
		sopID,
		actorUserID,
		map[string]string{
			"tag_id": tagID,
		},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit()
}

// SetTagStatus retires (soft-deletes) or revives a global tag.
// Inactive tags should be hidden from UI dropdowns, but remain visible on historical SOPs.
func (s *Service) SetTagStatus(tagID string, isActive bool, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setTagActiveStatusRecord(tx, tagID, isActive); err != nil {
		return fmt.Errorf("failed to update tag status: %w", err)
	}

	// Determine the semantic event name for the audit log
	eventName := audit.EventTagRetired
	if isActive {
		eventName = audit.EventTagRevived
	}

	err = s.auditLogger.Log(
		tx,
		eventName,
		audit.EntityTag,
		tagID,
		actorUserID,
		map[string]bool{
			"is_active": isActive,
		},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit()
}
