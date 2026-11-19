package sop

import (
	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

// AddAcknowledgment records a reader's sign-off on a published version and returns
// its id. Author acknowledgments are written when a version is created, approver
// acknowledgments only by ApproveSOPVersion, so any other type is refused.
func (s *Service) AddAcknowledgment(
	sopVersionID string,
	userID string,
	ackType string,
) (string, error) {
	if ackType != AckTypeRead {
		return "", errkind.New(errkind.Conflict, "only reader acknowledgments can be recorded directly")
	}
	result, err := s.runTransition(transitionRequest{
		transition:  transitionReaderAck,
		versionID:   sopVersionID,
		actorUserID: userID,
	})
	if err != nil {
		return "", err
	}
	return result.ackID, nil
}

// GetAcknowledgmentsByVersion returns all acks for a specific SOP version
func (s *Service) GetAcknowledgmentsByVersion(versionID string) ([]SOPAcknowledgment, error) {
	return getSOPAcknowledgmentsBySOPVersionIDRecord(s.db, versionID)
}

// GetAcknowledgmentsByVersionWithUser returns all acks for a specific SOP version with User Structs
func (s *Service) GetAcknowledgmentsByVersionWithUser(versionID string) ([]SOPAcknowledgmentWithUser, error) {
	return getSOPAcknowledgmentsWithUserBySOPVersionIDRecord(s.db, versionID)
}

// GetAcknowledgmentsByUser returns all acks made by a specific user
func (s *Service) GetAcknowledgmentsByUser(userID string) ([]SOPAcknowledgment, error) {
	return getSOPAcknowledgmentsByUserIDRecord(s.db, userID)
}

// GetSignatureStatusByUser returns the status of all published SOPs for a user
func (s *Service) GetSignatureStatusByUser(userID string) ([]UserSignatureStatus, error) {
	return getSignatureStatusByUserIDRecord(s.db, userID)
}
