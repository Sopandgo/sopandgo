package sop

import (
	"fmt"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

func expectedReaders(users []auth.User) []auth.User {
	out := make([]auth.User, 0, len(users))
	for _, user := range users {
		if !user.IsActive {
			continue
		}
		if auth.HasPermission(user.Role, auth.ScopeSOPSignReader) {
			out = append(out, user)
		}
	}
	return out
}

func coverageForPublished(row publishedVersionRow, readers []auth.User, acks map[string]string) SOPTrainingCoverage {
	signed := []TrainingMember{}
	unsigned := []TrainingMember{}
	for _, user := range readers {
		if signedAt, ok := acks[user.ID]; ok {
			at := signedAt
			signed = append(signed, TrainingMember{
				UserID:      user.ID,
				DisplayName: user.DisplayName,
				SignedAt:    &at,
			})
			continue
		}
		unsigned = append(unsigned, TrainingMember{
			UserID:      user.ID,
			DisplayName: user.DisplayName,
		})
	}
	return SOPTrainingCoverage{
		SOPID:        row.SOPID,
		Title:        row.Title,
		HasPublished: true,
		VersionID:    row.VersionID,
		Version:      row.Version,
		Signed:       signed,
		Unsigned:     unsigned,
	}
}

// TrainingCoverage lists reader-signature gaps for currently published SOPs.
// sopID empty returns every published SOP. A specific SOP with no published version
// returns a single row with HasPublished false.
func (s *Service) TrainingCoverage(sopID string, users []auth.User) ([]SOPTrainingCoverage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	readers := expectedReaders(users)
	rows, err := listCurrentlyPublishedVersionsRecord(s.db, sopID)
	if err != nil {
		return nil, fmt.Errorf("failed to list published versions: %w", err)
	}
	if sopID != "" && len(rows) == 0 {
		title := ""
		sopRow, err := getSOPByIDRecord(s.db, sopID)
		if err != nil {
			return nil, err
		}
		title = sopRow.Title
		return []SOPTrainingCoverage{{
			SOPID:        sopID,
			Title:        title,
			HasPublished: false,
			Signed:       []TrainingMember{},
			Unsigned:     []TrainingMember{},
		}}, nil
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.VersionID)
	}
	acks, err := listReaderAcksByVersionIDsRecord(s.db, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list reader acknowledgments: %w", err)
	}

	out := make([]SOPTrainingCoverage, 0, len(rows))
	for _, row := range rows {
		out = append(out, coverageForPublished(row, readers, acks[row.VersionID]))
	}
	return out, nil
}
