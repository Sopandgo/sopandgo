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

	"github.com/google/uuid"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
	"github.com/sopandgo/sopandgo/backend/internal/markdown"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
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

// RegisterSOPVersion saves new content as a draft authored by actorUserID and
// writes the author acknowledgment. Saving the same content as the latest version
// returns that version and writes nothing.
func (s *Service) RegisterSOPVersion(
	sopID string,
	content string,
	changeSummary string,
	actorUserID string,
) (string, int, error) {
	contentHash := integrity.HashBytes([]byte(content))

	// The content file is written inside the transaction; remove it again unless
	// the transaction commits.
	var writtenPath string
	writeVersion := func(tx *sql.Tx) (newVersion, error) {
		latestVersion := 0
		latest, err := getLatestSOPVersionRecord(tx, sopID)
		if err != nil && err != sql.ErrNoRows {
			return newVersion{}, fmt.Errorf("failed to fetch latest version: %w", err)
		}
		if err == nil {
			// Idempotency: the same content as the latest version is that version.
			if latest.ContentHash == contentHash {
				return newVersion{id: latest.ID, number: latest.Version, reused: true}, nil
			}
			latestVersion = latest.Version
		}

		summary, err := normalizeChangeSummary(changeSummary)
		if err != nil {
			return newVersion{}, err
		}
		if err := s.validateSOPContentAssets(sopID, content); err != nil {
			return newVersion{}, err
		}
		// Markdown policy (raw HTML, URL schemes) — authoritative for all API clients.
		if err := markdown.Validate(content); err != nil {
			return newVersion{}, err
		}

		number := latestVersion + 1
		id := uuid.NewString()
		createdAt := time.Now().UTC().Format(time.RFC3339Nano)

		// The database stores the relative path ("sops/<id>/version-1.md"); the
		// file is written at the absolute path under DATA_DIR.
		relPath := filepath.Join("sops", sopID, fmt.Sprintf("version-%d.md", number))
		absPath, err := storage.SafeJoin(s.dataDir, relPath)
		if err != nil {
			return newVersion{}, fmt.Errorf("invalid path construction: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return newVersion{}, fmt.Errorf("failed to create directory: %w", err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return newVersion{}, fmt.Errorf("failed to write file: %w", err)
		}
		writtenPath = absPath

		err = createSOPVersionRecord(tx, id, sopID, strconv.Itoa(number), filepath.ToSlash(relPath), contentHash, summary, createdAt)
		if err != nil {
			return newVersion{}, fmt.Errorf("failed to create version record: %w", err)
		}
		return newVersion{
			id:     id,
			number: number,
			auditPayload: map[string]any{
				"version":        number,
				"path":           relPath,
				"change_summary": summary,
			},
		}, nil
	}

	result, err := s.runTransition(transitionRequest{
		transition:   transitionCreate,
		sopID:        sopID,
		actorUserID:  actorUserID,
		writeVersion: writeVersion,
	})
	if err != nil {
		if writtenPath != "" {
			os.Remove(writtenPath)
		}
		return "", 0, err
	}
	return result.versionID, result.versionNumber, nil
}

func (s *Service) GetSOPVersionByID(id string) (*SOPVersion, error) {
	return getSOPVersionByIDRecord(s.db, id)
}

func (s *Service) GetSOPVersionsBySOPID(sopID string) ([]SOPVersion, error) {
	return getSOPVersionsBySOPIDRecord(s.db, sopID)
}

func (s *Service) GetSOPVersionIDLatestPublished(sopID string) (string, error) {
	// A plain query: a transaction would take the write lock for a read.
	latestPublishedVersion, err := getActivePublishedVersionIDRecord(s.db, sopID, "exclude-none")
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check for older published versions: %w", err)
	}

	return latestPublishedVersion, nil
}

func (s *Service) GetSOPVersionSummaryByID(id string) (*SOPVersionSummary, error) {
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

	var rejection *VersionRejection
	if v.Status == StateRejected {
		rejection, err = getVersionRejectionRecord(s.db, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load rejection: %w", err)
		}
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
		Rejection:       rejection,
	}

	return summary, nil
}

// VerifyVersionIntegrity checks if the version file on disk matches the DB hash
func (s *Service) VerifyVersionIntegrity(versionID string) (bool, error) {
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
	// 1. Get metadata from DB using the ID
	v, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return "", err
	}

	// 2. Resolve secure absolute path
	return storage.SafeJoin(s.dataDir, v.ContentPath)
}

// PromoteSOPVersion makes a draft the SOP's release candidate.
func (s *Service) PromoteSOPVersion(versionID string, actorUserID string) error {
	_, err := s.runTransition(transitionRequest{
		transition:  transitionPromote,
		versionID:   versionID,
		actorUserID: actorUserID,
	})
	return err
}

// RejectSOPVersion rejects a release candidate and stores the reason on the state
// row and in the audit log, so the version page and an audit can show it later.
func (s *Service) RejectSOPVersion(versionID string, actorUserID string, reason string) error {
	_, err := s.runTransition(transitionRequest{
		transition:  transitionReject,
		versionID:   versionID,
		actorUserID: actorUserID,
		reason:      reason,
	})
	return err
}

// ApproveSOPVersion publishes a release candidate with the approver's acknowledgment
// and supersedes the SOP's previously published version, in one transaction.
// It returns the approver acknowledgment's id.
func (s *Service) ApproveSOPVersion(versionID string, approverUserID string) (string, error) {
	result, err := s.runTransition(transitionRequest{
		transition:  transitionApprove,
		versionID:   versionID,
		actorUserID: approverUserID,
	})
	if err != nil {
		return "", err
	}
	return result.ackID, nil
}
