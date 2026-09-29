package sop_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestPublishLoop_SummaryDiffActivityAndTraining(t *testing.T) {
	env := testenv.New(t)
	editorID := "loop-editor"
	viewerID := "loop-viewer"
	auditorID := "loop-auditor"
	env.SeedTestUser(t, editorID, "editor@loop.local", auth.RoleEditor)
	env.SeedTestUser(t, viewerID, "viewer@loop.local", auth.RoleViewer)
	env.SeedTestUser(t, auditorID, "auditor@loop.local", auth.RoleAuditor)

	sopID, err := env.SOPService.RegisterSOP("Loop SOP", &editorID)
	if err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}

	_, _, err = env.SOPService.RegisterSOPVersion(sopID, "# One", "   ", &editorID)
	if !errors.Is(err, sop.ErrChangeSummaryRequired) {
		t.Fatalf("expected required summary, got %v", err)
	}

	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# One\nStep A\n", "First published procedure", &editorID)
	if err != nil {
		t.Fatalf("v1: %v", err)
	}
	if err := env.SOPService.TransitionVersionState(v1ID, sop.StateRC, editorID); err != nil {
		t.Fatalf("promote v1: %v", err)
	}
	if _, err := env.SOPService.ApproveSOPVersion(v1ID, editorID); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	v2ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# One\nStep A\nStep B\n", "Add step B", &editorID)
	if err != nil {
		t.Fatalf("v2: %v", err)
	}

	got, err := env.SOPService.GetSOPVersionByID(v2ID)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}
	if got.ChangeSummary != "Add step B" {
		t.Fatalf("summary %q", got.ChangeSummary)
	}

	diff, err := env.SOPService.DiffSOPVersion(sopID, v2ID, "")
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !diff.Comparable || diff.FromVersionID != v1ID {
		t.Fatalf("diff base: %+v", diff)
	}
	var added bool
	for _, line := range diff.Lines {
		if line.Kind == "add" && strings.Contains(line.Text, "Step B") {
			added = true
		}
	}
	if !added {
		t.Fatalf("expected added step B, lines=%+v", diff.Lines)
	}

	activity, err := env.SOPService.ListRecentPublishes(10)
	if err != nil {
		t.Fatalf("activity: %v", err)
	}
	if len(activity) != 1 || activity[0].VersionID != v1ID || activity[0].ChangeSummary != "First published procedure" {
		t.Fatalf("activity: %+v", activity)
	}

	users, err := env.AuthService.ListUsers()
	if err != nil {
		t.Fatalf("users: %v", err)
	}
	coverage, err := env.SOPService.TrainingCoverage(sopID, users)
	if err != nil {
		t.Fatalf("training: %v", err)
	}
	if len(coverage) != 1 || !coverage[0].HasPublished {
		t.Fatalf("coverage: %+v", coverage)
	}
	if len(coverage[0].Unsigned) != 2 {
		t.Fatalf("expected editor and viewer unsigned, got %+v", coverage[0].Unsigned)
	}
	for _, member := range coverage[0].Unsigned {
		if member.UserID == auditorID {
			t.Fatalf("auditor should not be expected to reader-sign")
		}
	}

	if _, err := env.SOPService.AddAcknowledgment(v1ID, viewerID, sop.AckTypeRead); err != nil {
		t.Fatalf("reader ack: %v", err)
	}
	coverage, err = env.SOPService.TrainingCoverage("", users)
	if err != nil {
		t.Fatalf("lab coverage: %v", err)
	}
	if len(coverage) != 1 || len(coverage[0].Signed) != 1 || coverage[0].Signed[0].UserID != viewerID {
		t.Fatalf("after sign: %+v", coverage)
	}
}
