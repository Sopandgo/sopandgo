package sop

import (
	"fmt"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/google/uuid"
)

func (s *Service) AddAcknowledgment(
	sopVersionID string,
	userID string,
	ackType string,
) (string, error) {
	// Author acks are written when a version is created. Approver acks are written
	// only by ApproveSOPVersion. This path is reader sign-off on a published version.
	if ackType != AckTypeRead {
		return "", fmt.Errorf("conflict: only reader acknowledgments can be recorded directly")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	v, err := getSOPVersionByIDRecord(tx, sopVersionID)
	if err != nil {
		return "", err
	}
	if v.Status != StatePublished {
		return "", fmt.Errorf("conflict: reader acknowledgments require a published version, current state is '%s'", v.Status)
	}

	// 1. Generate ID and Timestamp
	id := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 2. Create DB Record using Store function
	err = createSOPAcknowledgmentRecord(
		tx,
		id,
		sopVersionID,
		userID,
		ackType,
		createdAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create acknowledgment record: %w", err)
	}

	// 3. Audit Log
	err = s.auditLogger.Log(
		tx,
		audit.EventAcknowledgmentAdded,
		audit.EntityAcknowledgment,
		id,
		&userID,
		map[string]string{
			"sop_version_id": sopVersionID,
			"user_id":        userID,
			"ack_type":       ackType,
		},
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return id, nil
}

// GetAcknowledgmentsByVersion returns all acks for a specific SOP version
func (s *Service) GetAcknowledgmentsByVersion(versionID string) ([]SOPAcknowledgment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPAcknowledgmentsBySOPVersionIDRecord(s.db, versionID)
}

// GetAcknowledgmentsByVersionWithUser returns all acks for a specific SOP version with User Structs
func (s *Service) GetAcknowledgmentsByVersionWithUser(versionID string) ([]SOPAcknowledgmentWithUser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPAcknowledgmentsWithUserBySOPVersionIDRecord(s.db, versionID)
}

// GetAcknowledgmentsByUser returns all acks made by a specific user
func (s *Service) GetAcknowledgmentsByUser(userID string) ([]SOPAcknowledgment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPAcknowledgmentsByUserIDRecord(s.db, userID)
}

// GetSignatureStatusByUser returns the status of all published SOPs for a user
func (s *Service) GetSignatureStatusByUser(userID string) ([]UserSignatureStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSignatureStatusByUserIDRecord(s.db, userID)
}
