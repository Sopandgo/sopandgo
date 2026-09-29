package sop

import (
	"fmt"
	"os"

	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func (s *Service) readVersionContent(v *SOPVersion) (string, error) {
	absPath, err := storage.SafeJoin(s.dataDir, v.ContentPath)
	if err != nil {
		return "", fmt.Errorf("invalid storage path: %w", err)
	}
	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read sop content file: %w", err)
	}
	return string(contentBytes), nil
}

// DiffSOPVersion compares versionID to another version of the same SOP.
// againstVersionID empty selects the previous published version, or the previous version if none is published.
func (s *Service) DiffSOPVersion(sopID, versionID, againstVersionID string) (*VersionDiff, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	current, err := getSOPVersionByIDRecord(s.db, versionID)
	if err != nil {
		return nil, err
	}
	if current.SOPID != sopID {
		return nil, fmt.Errorf("sop version not found: %s", versionID)
	}

	baseID := againstVersionID
	if baseID == "" {
		baseID, err = previousComparisonVersionIDRecord(s.db, sopID, current.Version)
		if err != nil {
			return nil, err
		}
	}
	if baseID == "" || baseID == versionID {
		return &VersionDiff{
			Comparable:  false,
			ToVersionID: current.ID,
			ToVersion:   current.Version,
			Lines:       []DiffLine{},
		}, nil
	}

	base, err := getSOPVersionByIDRecord(s.db, baseID)
	if err != nil {
		return nil, err
	}
	if base.SOPID != sopID {
		return nil, fmt.Errorf("comparison version is not part of this SOP")
	}

	oldContent, err := s.readVersionContent(base)
	if err != nil {
		return nil, err
	}
	newContent, err := s.readVersionContent(current)
	if err != nil {
		return nil, err
	}

	return &VersionDiff{
		Comparable:    true,
		FromVersionID: base.ID,
		FromVersion:   base.Version,
		ToVersionID:   current.ID,
		ToVersion:     current.Version,
		Lines:         diffLines(splitLines(oldContent), splitLines(newContent)),
	}, nil
}

func (s *Service) ListRecentPublishes(limit int) ([]PublishedActivity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 12
	}
	if limit > 30 {
		limit = 30
	}
	return listRecentPublishesRecord(s.db, limit)
}
