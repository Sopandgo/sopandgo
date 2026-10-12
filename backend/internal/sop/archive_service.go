package sop

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// ArchiveSOP retires an SOP: it leaves the default list, signature status and team
// coverage, and its content, lifecycle and signatures are frozen until UnarchiveSOP.
// Nothing is deleted. A pending release candidate must be approved or rejected first,
// so no review is left waiting on a frozen SOP; drafts stay as they are.
func (s *Service) ArchiveSOP(sopID string, actorUserID string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrArchiveReasonRequired
	}
	if utf8.RuneCountInString(reason) > MaxArchiveReasonRunes {
		return ErrArchiveReasonTooLong
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := requireSOPActiveRecord(tx, sopID); err != nil {
		if err == ErrSOPArchived {
			return fmt.Errorf("conflict: this SOP is already archived")
		}
		return err
	}

	hasRC, err := hasReleaseCandidateForSOPRecord(tx, sopID)
	if err != nil {
		return fmt.Errorf("failed to check release candidates: %w", err)
	}
	if hasRC {
		return fmt.Errorf("conflict: approve or reject the pending release candidate before archiving this SOP")
	}

	archivedAt := time.Now().UTC().Format(time.RFC3339Nano)
	if err := setSOPArchivedRecord(tx, sopID, archivedAt, actorUserID, reason); err != nil {
		return fmt.Errorf("failed to archive sop: %w", err)
	}

	err = s.auditLogger.Log(
		tx,
		audit.EventSOPArchived,
		audit.EntitySOP,
		sopID,
		&actorUserID,
		map[string]string{
			"reason": reason,
		},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit()
}

// UnarchiveSOP puts an archived SOP back in use, exactly as it was archived.
func (s *Service) UnarchiveSOP(sopID string, actorUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = requireSOPActiveRecord(tx, sopID)
	if err == nil {
		return fmt.Errorf("conflict: this SOP is not archived")
	}
	if err != ErrSOPArchived {
		return err
	}

	if err := clearSOPArchivedRecord(tx, sopID); err != nil {
		return fmt.Errorf("failed to unarchive sop: %w", err)
	}

	err = s.auditLogger.Log(
		tx,
		audit.EventSOPUnarchived,
		audit.EntitySOP,
		sopID,
		&actorUserID,
		map[string]string{},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit()
}
