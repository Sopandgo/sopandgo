package sop

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/markdown"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
	"github.com/google/uuid"
)

func normalizeChangeSummary(raw string) (string, error) {
	summary := strings.TrimSpace(raw)
	summary = strings.Join(strings.Fields(summary), " ")
	if summary == "" {
		return "", ErrChangeSummaryRequired
	}
	if utf8.RuneCountInString(summary) > MaxChangeSummaryRunes {
		return "", ErrChangeSummaryTooLong
	}
	return summary, nil
}

func (s *Service) RegisterSOPVersion(
	sopID string,
	content string,
	changeSummary string,
	actorUserID *string,
) (string, int, error) {
	contentHash := integrity.HashBytes([]byte(content))

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback()

	// 1. Only one release candidate at a time per SOP; drafts may exist in parallel.
	hasRC, err := hasReleaseCandidateForSOPRecord(tx, sopID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to check release candidates: %w", err)
	}
	if hasRC {
		return "", 0, fmt.Errorf("conflict: cannot create a new version while a release candidate is pending approval for this SOP")
	}

	var (
		latestVersion int
		latestHash    string
		latestID      string
	)

	// 2. Get next version number & idempotency anchor (latest by version)
	latestV, err := getLatestSOPVersionRecord(tx, sopID)
	if err != nil {
		if err == sql.ErrNoRows {
			latestVersion = 0
			latestHash = ""
		} else {
			return "", 0, fmt.Errorf("failed to fetch latest version: %w", err)
		}
	} else {
		latestVersion = latestV.Version
		latestHash = latestV.ContentHash
		latestID = latestV.ID
	}

	// 3. IDEMPOTENCY CHECK
	if latestHash != "" && latestHash == contentHash {
		return latestID, latestVersion, nil
	}

	summary, err := normalizeChangeSummary(changeSummary)
	if err != nil {
		return "", 0, err
	}

	newVersion := latestVersion + 1

	// 4. Validate Assets
	if err := s.validateSOPContentAssets(sopID, content); err != nil {
		return "", 0, err
	}

	// 5. Markdown policy (raw HTML, URL schemes) — authoritative for all API clients.
	if err := markdown.Validate(content); err != nil {
		return "", 0, err
	}

	id := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 6. Prepare Paths
	filename := fmt.Sprintf("version-%d.md", newVersion)

	// DB PATH: Relative (Portable) -> "sops/<id>/version-1.md"
	relPath := filepath.Join("sops", sopID, filename)

	dbPath := filepath.ToSlash(relPath)

	// DISK PATH: Absolute (Secure) -> "/data/sops/<id>/version-1.md"
	absPath, err := storage.SafeJoin(s.dataDir, relPath)
	if err != nil {
		return "", 0, fmt.Errorf("invalid path construction: %w", err)
	}

	// Ensure directory exists using the absolute path
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// 7. Write File to Disk using Absolute Path
	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return "", 0, fmt.Errorf("failed to write file: %w", err)
	}

	// Cleanup: Delete file if DB insert fails
	commitSuccessful := false
	defer func() {
		if !commitSuccessful {
			os.Remove(absPath)
		}
	}()

	// 8. Create DB Record using db Path
	err = createSOPVersionRecord(
		tx,
		id,
		sopID,
		strconv.Itoa(newVersion),
		dbPath,
		contentHash,
		summary,
		createdAt,
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create version record: %w", err)
	}

	// 10. Audit Log for Version Creation
	err = s.auditLogger.Log(
		tx,
		audit.EventSOPVersionCreated,
		audit.EntitySOPVersion,
		id,
		actorUserID,
		map[string]any{
			"version":        newVersion,
			"path":           relPath,
			"change_summary": summary,
		},
	)
	if err != nil {
		return "", 0, fmt.Errorf("audit log failed: %w", err)
	}

	// 11. INITIALIZE LIFECYCLE STATE AS 'draft'
	actorID := ""
	if actorUserID != nil {
		actorID = *actorUserID
	}

	stateID := uuid.NewString()
	err = createSOPVersionStateRecord(tx, stateID, id, StateDraft, actorID, createdAt)
	if err != nil {
		return "", 0, fmt.Errorf("failed to initialize version state: %w", err)
	}

	// 12. RECORD AUTHOR ACKNOWLEDGMENT
	if actorID != "" {
		ackID := uuid.NewString()
		// Assuming AckTypeAuthor is defined in your constants!
		err = createSOPAcknowledgmentRecord(tx, ackID, id, actorID, AckTypeAuthor, createdAt)
		if err != nil {
			return "", 0, fmt.Errorf("failed to record author acknowledgment: %w", err)
		}

		// Audit log the acknowledgment
		err = s.auditLogger.Log(tx, audit.EventSOPAcknowledgmentCreated, audit.EntitySOPVersion, id, actorUserID, map[string]any{
			"ack_type": AckTypeAuthor,
			"ack_id":   ackID,
		})
		if err != nil {
			return "", 0, fmt.Errorf("audit log failed for author acknowledgment: %w", err)
		}
	}

	// 13. Commit
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}

	commitSuccessful = true
	if actorID != "" {
		s.generateVersionPDFArtifactAfterCommit(id, StateDraft, actorID)
	}
	return id, newVersion, nil
}

func (s *Service) GetSOPVersionByID(id string) (*SOPVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPVersionByIDRecord(s.db, id)
}

func (s *Service) GetSOPVersionsBySOPID(sopID string) ([]SOPVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getSOPVersionsBySOPIDRecord(s.db, sopID)
}

func (s *Service) GetSOPVersionIDLatestPublished(sopID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	latestPublishedVersion, err := getActivePublishedVersionIDRecord(tx, sopID, "exclude-none")
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check for older published versions: %w", err)
	}

	return latestPublishedVersion, nil
}

func (s *Service) GetSOPVersionSummaryByID(id string) (*SOPVersionSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, err := getSOPVersionByIDRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	assets, err := getSOPAssetsBySOPIDRecord(s.db, v.SOPID)
	if err != nil {
		return nil, err
	}

	tags, err := getTagsForSOPRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	acks, err := getSOPAcknowledgmentsWithUserBySOPVersionIDRecord(s.db, id)
	if err != nil {
		return nil, err
	}

	absPath, err := storage.SafeJoin(s.dataDir, v.ContentPath)
	if err != nil {
		return nil, fmt.Errorf("invalid storage path: %w", err)
	}

	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sop content file: %w", err)
	}

	hashValid, err := integrity.VerifyFile(absPath, v.ContentHash)
	if err != nil {
		return nil, fmt.Errorf("integrity check failed: %w", err)
	}

	summary := &SOPVersionSummary{
		ID:              v.ID,
		SOPID:           v.SOPID,
		Version:         v.Version,
		Status:          v.Status,
		ChangeSummary:   v.ChangeSummary,
		ContentHash:     v.ContentHash,
		Assets:          assets,
		Acknowledgments: acks,
		CreatedAt:       v.CreatedAt,
		Tags:            tags,
		Content:         string(contentBytes), // Convert bytes to string
		HashValid:       hashValid,
	}

	return summary, nil
}

// VerifyVersionIntegrity checks if the version file on disk matches the DB hash
func (s *Service) VerifyVersionIntegrity(versionID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Get metadata from DB
	v, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return false, err
	}

	// 2. Resolve secure path
	absPath, err := storage.SafeJoin(s.dataDir, v.ContentPath)
	if err != nil {
		return false, fmt.Errorf("invalid storage path: %w", err)
	}

	// 3. Verify Hash using your integrity package
	return integrity.VerifyFile(absPath, v.ContentHash)
}

// GetVersionPath resolves the secure absolute path for a version file.
func (s *Service) GetVersionPath(versionID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Get metadata from DB using the ID
	v, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return "", err
	}

	// 2. Resolve secure absolute path
	return storage.SafeJoin(s.dataDir, v.ContentPath)
}

// validateManualVersionTransition enforces the allowed API-driven lifecycle transitions.
// Publishing and superseding are reserved for ApproveSOPVersion.
func validateManualVersionTransition(currentState, newState string) error {
	if newState == StatePublished || newState == StateSuperseded {
		return fmt.Errorf("conflict: %q and %q must be applied via ApproveSOPVersion, not TransitionVersionState", StatePublished, StateSuperseded)
	}
	switch currentState {
	case StateDraft:
		if newState != StateRC && newState != StateRejected {
			return fmt.Errorf("conflict: invalid transition from %s to %s (allowed: %s or %s)", currentState, newState, StateRC, StateRejected)
		}
	case StateRC:
		if newState != StateRejected {
			return fmt.Errorf("conflict: invalid transition from %s to %s (use ApproveSOPVersion to publish)", currentState, newState)
		}
	case StatePublished, StateRejected, StateSuperseded:
		return fmt.Errorf("conflict: cannot transition from terminal state %q", currentState)
	default:
		if currentState == "" {
			return fmt.Errorf("conflict: version has no resolvable lifecycle state")
		}
		return fmt.Errorf("conflict: unknown current state %q", currentState)
	}
	return nil
}

// TransitionVersionState appends a new lifecycle state to a version.
func (s *Service) TransitionVersionState(versionID string, newState string, actorUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Verify the version actually exists and enforce the state machine
	v, err := getSOPVersionByIDRecord(tx, versionID)
	if err != nil {
		return fmt.Errorf("could not find version: %w", err)
	}
	if err := validateManualVersionTransition(v.Status, newState); err != nil {
		return err
	}

	if newState == StateRC {
		other, err := hasAnotherReleaseCandidateForSOPRecord(tx, v.SOPID, versionID)
		if err != nil {
			return fmt.Errorf("failed to check release candidates: %w", err)
		}
		if other {
			return fmt.Errorf("conflict: a release candidate already exists for this SOP; resolve it before promoting another version")
		}
	}

	// 2. Create the new state record
	stateID := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	err = createSOPVersionStateRecord(tx, stateID, versionID, newState, actorUserID, createdAt)
	if err != nil {
		return fmt.Errorf("failed to append new state: %w", err)
	}

	// 3. Cryptographic Audit Log
	err = s.auditLogger.Log(
		tx,
		audit.EventSOPVersionStateChanged,
		audit.EntitySOPVersion,
		versionID,
		&actorUserID,
		map[string]any{
			"new_state": newState,
		},
	)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	s.generateVersionPDFArtifactAfterCommit(versionID, newState, actorUserID)
	return nil
}

// ApproveSOPVersion records an approver's signature and promotes the version to 'published' in a single transaction.
// It also automatically supersedes any previously published version for the same SOP.
func (s *Service) ApproveSOPVersion(versionID string, approverUserID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// 1. Verify the version exists and is currently an RC
	v, err := getSOPVersionByIDRecord(tx, versionID)
	if err != nil {
		return "", fmt.Errorf("could not find version: %w", err)
	}
	if v.Status != StateRC {
		return "", fmt.Errorf("conflict: only release candidates can be approved, current state is '%s'", v.Status)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	ackID := uuid.NewString()
	stateID := uuid.NewString()

	// 2. Supersede any previously published version for this SOP *before* inserting the new
	//    published state so the DB never temporarily has two concurrently published versions.
	oldPublishedVersionID, err := getActivePublishedVersionIDRecord(tx, v.SOPID, versionID)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check for older published versions: %w", err)
	}

	if oldPublishedVersionID != "" {
		supersedeStateID := uuid.NewString()

		err = createSOPVersionStateRecord(tx, supersedeStateID, oldPublishedVersionID, StateSuperseded, approverUserID, now)
		if err != nil {
			return "", fmt.Errorf("failed to supersede previous version: %w", err)
		}

		err = s.auditLogger.Log(tx, audit.EventSOPVersionStateChanged, audit.EntitySOPVersion, oldPublishedVersionID, &approverUserID, map[string]any{
			"new_state": StateSuperseded,
			"reason":    fmt.Sprintf("Superseded by version %d", v.Version),
		})
		if err != nil {
			return "", fmt.Errorf("audit log failed for superseding old version: %w", err)
		}
	}

	// 3. Record the Signature (Acknowledgment)
	err = createSOPAcknowledgmentRecord(tx, ackID, versionID, approverUserID, AckTypeApproved, now)
	if err != nil {
		return "", fmt.Errorf("failed to create acknowledgment record: %w", err)
	}

	// 4. Append the new Lifecycle State ('published') for the approved version
	err = createSOPVersionStateRecord(tx, stateID, versionID, StatePublished, approverUserID, now)
	if err != nil {
		return "", fmt.Errorf("failed to append published state: %w", err)
	}

	// 5. Cryptographic Audit Logs (for the newly approved version)
	err = s.auditLogger.Log(tx, audit.EventSOPAcknowledgmentCreated, audit.EntitySOPVersion, versionID, &approverUserID, map[string]any{
		"ack_type": AckTypeApproved,
		"ack_id":   ackID,
	})
	if err != nil {
		return "", fmt.Errorf("audit log failed for acknowledgment: %w", err)
	}

	err = s.auditLogger.Log(tx, audit.EventSOPVersionStateChanged, audit.EntitySOPVersion, versionID, &approverUserID, map[string]any{
		"new_state": StatePublished,
	})
	if err != nil {
		return "", fmt.Errorf("audit log failed for state change: %w", err)
	}

	// 6. Commit the Atomic Transaction
	if err := tx.Commit(); err != nil {
		return "", err
	}
	s.generateVersionPDFArtifactAfterCommit(versionID, StatePublished, approverUserID)

	return ackID, nil
}
