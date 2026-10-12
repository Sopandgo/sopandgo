package sop_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// publishedSOP creates an SOP whose v1 is published and returns the SOP and version ids.
func publishedSOP(t *testing.T, env *testenv.Env, title, actorID string) (string, string) {
	t.Helper()
	sopID, err := env.SOPService.RegisterSOP(title, &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}
	vID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# "+title+"\n", "First version", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOPVersion: %v", err)
	}
	if err := env.SOPService.TransitionVersionState(vID, sop.StateRC, actorID); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if _, err := env.SOPService.ApproveSOPVersion(vID, actorID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	return sopID, vID
}

func TestService_ArchiveSOP_ReasonAndConflicts(t *testing.T) {
	env := testenv.New(t)
	approverID := "archive-approver"
	env.SeedTestUser(t, approverID, "approver@archive.local", auth.RoleApprover)

	sopID, _ := publishedSOP(t, env, "Old centrifuge", approverID)

	if err := env.SOPService.ArchiveSOP(sopID, approverID, "   "); !errors.Is(err, sop.ErrArchiveReasonRequired) {
		t.Fatalf("blank reason: got %v", err)
	}
	if err := env.SOPService.ArchiveSOP(sopID, approverID, strings.Repeat("x", sop.MaxArchiveReasonRunes+1)); !errors.Is(err, sop.ErrArchiveReasonTooLong) {
		t.Fatalf("long reason: got %v", err)
	}
	if err := env.SOPService.ArchiveSOP("missing-sop", approverID, "Gone"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing SOP: got %v", err)
	}
	if err := env.SOPService.UnarchiveSOP(sopID, approverID); err == nil || !strings.HasPrefix(err.Error(), "conflict:") {
		t.Fatalf("unarchive active SOP: got %v", err)
	}

	if err := env.SOPService.ArchiveSOP(sopID, approverID, "  Centrifuge replaced  "); err != nil {
		t.Fatalf("ArchiveSOP: %v", err)
	}
	if err := env.SOPService.ArchiveSOP(sopID, approverID, "Again"); err == nil || !strings.HasPrefix(err.Error(), "conflict:") {
		t.Fatalf("archive twice: got %v", err)
	}

	detail, err := env.SOPService.GetSOPByIDWithTags(sopID, approverID)
	if err != nil {
		t.Fatalf("GetSOPByIDWithTags: %v", err)
	}
	if detail.Archive == nil {
		t.Fatal("expected archive details")
	}
	if detail.Archive.Reason != "Centrifuge replaced" || detail.Archive.ActorUserID != approverID || detail.Archive.ActorName != "Test User" {
		t.Fatalf("unexpected archive details: %+v", detail.Archive)
	}
	if detail.Archive.ArchivedAt.IsZero() {
		t.Fatal("expected archived_at")
	}

	var payload string
	if err := env.Store.DB.QueryRow(
		`SELECT payload FROM audit_events WHERE event_type = 'sop_archived' AND entity_id = ?`, sopID,
	).Scan(&payload); err != nil {
		t.Fatalf("audit event: %v", err)
	}
	if !strings.Contains(payload, "Centrifuge replaced") {
		t.Fatalf("audit payload %q lacks the reason", payload)
	}

	if err := env.SOPService.UnarchiveSOP(sopID, approverID); err != nil {
		t.Fatalf("UnarchiveSOP: %v", err)
	}
	detail, _ = env.SOPService.GetSOPByIDWithTags(sopID, approverID)
	if detail.Archive != nil {
		t.Fatalf("expected no archive after unarchive, got %+v", detail.Archive)
	}
	var n int
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'sop_unarchived' AND entity_id = ?`, sopID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected one sop_unarchived event, got %d", n)
	}
}

func TestService_ArchiveSOP_BlockedByReleaseCandidate(t *testing.T) {
	env := testenv.New(t)
	approverID := "archive-rc-approver"
	env.SeedTestUser(t, approverID, "rc@archive.local", auth.RoleApprover)

	sopID, _ := publishedSOP(t, env, "Pending review", approverID)
	v2ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Pending review\nMore\n", "Second", &approverID)
	if err != nil {
		t.Fatalf("v2: %v", err)
	}
	if err := env.SOPService.TransitionVersionState(v2ID, sop.StateRC, approverID); err != nil {
		t.Fatalf("promote v2: %v", err)
	}

	if err := env.SOPService.ArchiveSOP(sopID, approverID, "Retired"); err == nil || !strings.HasPrefix(err.Error(), "conflict:") {
		t.Fatalf("expected conflict with a pending RC, got %v", err)
	}

	if err := env.SOPService.RejectSOPVersion(v2ID, approverID, "Not needed"); err != nil {
		t.Fatalf("reject v2: %v", err)
	}
	if err := env.SOPService.ArchiveSOP(sopID, approverID, "Retired"); err != nil {
		t.Fatalf("archive after rejecting the RC: %v", err)
	}
}

func TestService_ArchivedSOPIsReadOnly(t *testing.T) {
	env := testenv.New(t)
	approverID := "archive-ro-approver"
	readerID := "archive-ro-reader"
	env.SeedTestUser(t, approverID, "ro@archive.local", auth.RoleApprover)
	env.SeedTestUser(t, readerID, "reader@archive.local", auth.RoleViewer)

	sopID, publishedID := publishedSOP(t, env, "Frozen SOP", approverID)
	draftID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Frozen SOP\nDraft\n", "Draft change", &approverID)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}

	if err := env.SOPService.ArchiveSOP(sopID, approverID, "No longer used"); err != nil {
		t.Fatalf("ArchiveSOP: %v", err)
	}

	checks := map[string]error{}
	_, _, checks["new version"] = env.SOPService.RegisterSOPVersion(sopID, "# Frozen SOP\nNew\n", "New", &approverID)
	checks["promote"] = env.SOPService.TransitionVersionState(draftID, sop.StateRC, approverID)
	checks["reject"] = env.SOPService.RejectSOPVersion(draftID, approverID, "No")
	_, checks["approve"] = env.SOPService.ApproveSOPVersion(draftID, approverID)
	_, checks["reader signature"] = env.SOPService.AddAcknowledgment(publishedID, readerID, sop.AckTypeRead)
	_, checks["asset"] = env.SOPService.AddAsset(sopID, "diagram.png", []byte("png"), &approverID)
	for name, err := range checks {
		if !errors.Is(err, sop.ErrSOPArchived) {
			t.Errorf("%s: expected ErrSOPArchived, got %v", name, err)
		}
	}

	// Reading still works
	if _, err := env.SOPService.GetSOPVersionSummaryByID(publishedID); err != nil {
		t.Fatalf("read archived version: %v", err)
	}

	// Unarchiving restores every write
	if err := env.SOPService.UnarchiveSOP(sopID, approverID); err != nil {
		t.Fatalf("UnarchiveSOP: %v", err)
	}
	if _, err := env.SOPService.AddAcknowledgment(publishedID, readerID, sop.AckTypeRead); err != nil {
		t.Fatalf("reader signature after unarchive: %v", err)
	}
	if err := env.SOPService.TransitionVersionState(draftID, sop.StateRC, approverID); err != nil {
		t.Fatalf("promote after unarchive: %v", err)
	}
}

func TestService_ArchivedSOPLeavesListsAndSignatures(t *testing.T) {
	env := testenv.New(t)
	approverID := "archive-list-approver"
	readerID := "archive-list-reader"
	env.SeedTestUser(t, approverID, "list@archive.local", auth.RoleApprover)
	env.SeedTestUser(t, readerID, "listreader@archive.local", auth.RoleViewer)

	activeID, _ := publishedSOP(t, env, "Active SOP", approverID)
	archivedID, _ := publishedSOP(t, env, "Archived SOP", approverID)
	if err := env.SOPService.FavoriteSOP(archivedID, readerID); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if err := env.SOPService.ArchiveSOP(archivedID, approverID, "Retired"); err != nil {
		t.Fatalf("ArchiveSOP: %v", err)
	}

	ids := func(items []sop.SOPListItem) []string {
		out := []string{}
		for _, it := range items {
			out = append(out, it.ID)
		}
		return out
	}

	active, total, err := env.SOPService.ListSOPs(readerID, 50, 0, "", "", false, false, sop.ArchiveFilterActive)
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if total != 1 || len(active) != 1 || active[0].ID != activeID || active[0].ArchivedAt != nil {
		t.Fatalf("active list: total %d, ids %v", total, ids(active))
	}

	only, total, _ := env.SOPService.ListSOPs(readerID, 50, 0, "", "", false, false, sop.ArchiveFilterOnly)
	if total != 1 || len(only) != 1 || only[0].ID != archivedID || only[0].ArchivedAt == nil {
		t.Fatalf("archived list: total %d, ids %v", total, ids(only))
	}

	_, total, _ = env.SOPService.ListSOPs(readerID, 50, 0, "", "", false, false, sop.ArchiveFilterAll)
	if total != 2 {
		t.Fatalf("all list: total %d", total)
	}

	// The dashboard's favorites use the default (active) list
	favs, total, _ := env.SOPService.ListSOPs(readerID, 50, 0, "", "", true, false, sop.ArchiveFilterActive)
	if total != 0 || len(favs) != 0 {
		t.Fatalf("archived favorite still listed: %v", ids(favs))
	}

	statuses, err := env.SOPService.GetSignatureStatusByUser(readerID)
	if err != nil {
		t.Fatalf("signature status: %v", err)
	}
	if len(statuses) != 1 || statuses[0].SOPID != activeID {
		t.Fatalf("signature status should list only the active SOP, got %+v", statuses)
	}

	users := []auth.User{{ID: readerID, DisplayName: "Reader", Role: auth.RoleViewer, IsActive: true}}
	coverage, err := env.SOPService.TrainingCoverage("", users)
	if err != nil {
		t.Fatalf("training coverage: %v", err)
	}
	if len(coverage) != 1 || coverage[0].SOPID != activeID {
		t.Fatalf("team coverage should list only the active SOP, got %+v", coverage)
	}
	// The SOP's own coverage stays available as history
	own, err := env.SOPService.TrainingCoverage(archivedID, users)
	if err != nil || len(own) != 1 || !own[0].HasPublished {
		t.Fatalf("archived SOP coverage: %+v, %v", own, err)
	}
}
