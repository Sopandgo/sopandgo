package sop_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_TransitionVersionState_Validation(t *testing.T) {
	env := testenv.New(t)
	actorID := "transition-guard-user"
	env.SeedTestUser(t, actorID, "transition-guard@demo.local", "admin")

	sopID, err := env.SOPService.RegisterSOP("Transition Guard SOP", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP failed: %v", err)
	}
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "Content", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOPVersion failed: %v", err)
	}

	err = env.SOPService.TransitionVersionState(v1ID, sop.StatePublished, actorID)
	if err == nil {
		t.Fatal("Expected error when transitioning draft directly to published")
	}
	if !strings.Contains(err.Error(), "conflict:") {
		t.Errorf("Expected conflict error, got: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("draft -> rc failed: %v", err)
	}

	err = env.SOPService.TransitionVersionState(v1ID, sop.StatePublished, actorID)
	if err == nil {
		t.Fatal("Expected error when transitioning rc to published via TransitionVersionState")
	}

	_, err = env.SOPService.ApproveSOPVersion(v1ID, actorID)
	if err != nil {
		t.Fatalf("ApproveSOPVersion failed: %v", err)
	}

	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err == nil {
		t.Fatal("Expected error when transitioning from published")
	}
}

func TestService_RegisterSOPVersion_Lifecycle(t *testing.T) {
	env := testenv.New(t)
	actorID := "editor-user-1"
	approverID := "approver-user-1"

	// FIX: Seed the users!
	env.SeedTestUser(t, actorID, "integrity@demo.local", "admin")
	env.SeedTestUser(t, approverID, "approver@demo.local", "admin")

	// 1. Setup: Create a base SOP container
	sopID, err := env.SOPService.RegisterSOP("Database Backup Procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to setup SOP: %v", err)
	}

	// 2. Register Version 1 (Happy Path)
	contentV1 := "# V1 Backup Procedure\nRun the script."
	v1ID, v1Num, err := env.SOPService.RegisterSOPVersion(sopID, contentV1, "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to register V1: %v", err)
	}
	if v1Num != 1 {
		t.Errorf("Expected version number 1, got %d", v1Num)
	}

	// Verify file exists on disk
	v1Path, _ := env.SOPService.GetVersionPath(v1ID)
	if _, err := os.Stat(v1Path); os.IsNotExist(err) {
		t.Errorf("Version file was not physically created on disk at %s", v1Path)
	}

	// 3. Parallel drafts: a new version may be registered while an older one is still draft.
	v2ParallelID, v2ParallelNum, err := env.SOPService.RegisterSOPVersion(sopID, "Parallel draft body", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Expected second draft while V1 is draft: %v", err)
	}
	if v2ParallelNum != 2 {
		t.Fatalf("Expected version 2, got %d", v2ParallelNum)
	}

	// 4. Transition State & Approve V1 (older version can move through RC while V2 stays draft)
	time.Sleep(15 * time.Millisecond) // <-- Added sleep
	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("Failed to transition to RC: %v", err)
	}

	time.Sleep(15 * time.Millisecond) // <-- Added sleep
	_, err = env.SOPService.ApproveSOPVersion(v1ID, approverID)
	if err != nil {
		t.Fatalf("Failed to approve V1: %v", err)
	}

	time.Sleep(15 * time.Millisecond) // <-- Added sleep to ensure V1 is fully recognized as published

	// 5. Register Version 3 (latest was V2 draft; V1 is now published)
	contentV2 := "# V2 Backup Procedure\nRun the updated script."
	v3ID, v3Num, err := env.SOPService.RegisterSOPVersion(sopID, contentV2, "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to register V3 after V1 was published: %v", err)
	}
	if v3Num != 3 {
		t.Errorf("Expected version number 3, got %d", v3Num)
	}
	_ = v2ParallelID // V2 remains an older draft alongside V3

	// 6. Test Idempotency
	// If we submit the exact same content for a new version, the system should catch the identical hash.
	time.Sleep(15 * time.Millisecond) // <-- Added sleep
	err = env.SOPService.TransitionVersionState(v3ID, sop.StateRejected, actorID)
	if err != nil {
		t.Fatalf("Failed to reject V2: %v", err)
	}

	time.Sleep(15 * time.Millisecond) // <-- Added sleep
	idempotentID, idempotentNum, err := env.SOPService.RegisterSOPVersion(sopID, contentV2, "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Idempotency check failed with error: %v", err)
	}
	if idempotentID != v3ID || idempotentNum != v3Num {
		t.Errorf("Idempotency failed. Expected to return V3 (%s, num %d), got (%s, num %d)", v3ID, v3Num, idempotentID, idempotentNum)
	}
}

func TestService_RegisterSOPVersion_RCBlocksParallelCreate(t *testing.T) {
	env := testenv.New(t)
	actorID := "rc-block-user"
	env.SeedTestUser(t, actorID, "rcblock@demo.local", "admin")

	sopID, err := env.SOPService.RegisterSOP("RC Block SOP", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP failed: %v", err)
	}
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "V1", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Register V1 failed: %v", err)
	}
	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("promote to RC failed: %v", err)
	}

	_, _, err = env.SOPService.RegisterSOPVersion(sopID, "V2 while V1 is RC", "Updated procedure", &actorID)
	if err == nil || !strings.Contains(err.Error(), "release candidate") {
		t.Fatalf("Expected conflict while RC exists, got: %v", err)
	}
}

func TestService_TransitionVersionState_SecondRCBlocked(t *testing.T) {
	env := testenv.New(t)
	actorID := "double-rc-user"
	env.SeedTestUser(t, actorID, "doublerc@demo.local", "admin")

	sopID, err := env.SOPService.RegisterSOP("Double RC SOP", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP failed: %v", err)
	}
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "A", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Register V1 failed: %v", err)
	}
	v2ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "B", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Register V2 failed: %v", err)
	}
	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("first RC promotion failed: %v", err)
	}
	err = env.SOPService.TransitionVersionState(v2ID, sop.StateRC, actorID)
	if err == nil || !strings.Contains(err.Error(), "release candidate already exists") {
		t.Fatalf("Expected conflict on second RC, got: %v", err)
	}
}

func TestService_ApproveSOPVersion_SupersedesOldVersions(t *testing.T) {
	env := testenv.New(t)
	actorID := "user-1"

	env.SeedTestUser(t, actorID, "integrity@demo.local", "admin")

	// 1. Setup
	sopID, err := env.SOPService.RegisterSOP("Fire Drill", &actorID)
	if err != nil {
		t.Fatalf("RegisterSOP failed: %v", err)
	}

	// Create and Publish V1
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "V1", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Register V1 failed: %v", err)
	}

	time.Sleep(15 * time.Millisecond) // Ensure distinct timestamp
	err = env.SOPService.TransitionVersionState(v1ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("Transition V1 to RC failed: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	_, err = env.SOPService.ApproveSOPVersion(v1ID, actorID)
	if err != nil {
		t.Fatalf("Approve V1 failed: %v", err)
	}

	time.Sleep(15 * time.Millisecond)

	// Create and Publish V2
	v2ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "V2", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Register V2 failed: %v", err) // <-- This is where your code was originally failing silently!
	}

	time.Sleep(15 * time.Millisecond)
	err = env.SOPService.TransitionVersionState(v2ID, sop.StateRC, actorID)
	if err != nil {
		t.Fatalf("Transition V2 to RC failed: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	_, err = env.SOPService.ApproveSOPVersion(v2ID, actorID)
	if err != nil {
		t.Fatalf("Approve V2 failed: %v", err)
	}

	// 2. Assert V1 was auto-superseded
	v1, err := env.SOPService.GetSOPVersionByID(v1ID)
	if err != nil {
		t.Fatalf("Failed to fetch V1: %v", err)
	}
	if v1.Status != sop.StateSuperseded {
		t.Errorf("Expected V1 to be auto-superseded, but status is %q", v1.Status)
	}

	// 3. Assert V2 is published
	v2, err := env.SOPService.GetSOPVersionByID(v2ID)
	if err != nil {
		t.Fatalf("Failed to fetch V2: %v", err)
	}
	if v2.Status != sop.StatePublished {
		t.Errorf("Expected V2 to be published, but status is %q", v2.Status)
	}

	// 4. Verify Latest Published Fetcher
	latestPubID, err := env.SOPService.GetSOPVersionIDLatestPublished(sopID)
	if err != nil {
		t.Fatalf("Failed to fetch latest published: %v", err)
	}
	if latestPubID != v2ID {
		t.Errorf("Expected latest published ID to be V2 (%s), got %s", v2ID, latestPubID)
	}
}

func TestService_GetSOPVersionSummaryByID(t *testing.T) {
	env := testenv.New(t)
	actorID := "user-1"

	// FIX: Seed the user!
	env.SeedTestUser(t, actorID, "integrity@demo.local", "admin")

	sopID, _ := env.SOPService.RegisterSOP("Summary Test", &actorID)
	content := "Test Content"
	v1ID, _, _ := env.SOPService.RegisterSOPVersion(sopID, content, "Updated procedure", &actorID)

	summary, err := env.SOPService.GetSOPVersionSummaryByID(v1ID)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	if summary.Status != sop.StateDraft {
		t.Errorf("Expected draft status, got %s", summary.Status)
	}
	if summary.Content != content {
		t.Errorf("Expected content %q, got %q", content, summary.Content)
	}
	if !summary.HashValid {
		t.Errorf("Expected HashValid to be true, but integrity check failed inside summary")
	}
	if len(summary.Acknowledgments) != 1 {
		t.Errorf("Expected exactly 1 acknowledgment (the author ack), got %d", len(summary.Acknowledgments))
	}
}

func TestService_RegisterSOPVersion_MissingAssetValidation(t *testing.T) {
	env := testenv.New(t)
	actorID := "user-1"

	env.SeedTestUser(t, actorID, "integrity@demo.local", "admin")

	sopID, _ := env.SOPService.RegisterSOP("Asset Validation Test", &actorID)

	// Attempt to register a version that links to an asset we haven't uploaded yet
	content := "Check out this image: ![diagram](assets/missing-diagram.png)"

	_, _, err := env.SOPService.RegisterSOPVersion(sopID, content, "Updated procedure", &actorID)

	if err == nil {
		t.Fatal("Expected asset validation to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "referenced asset not found: assets/missing-diagram.png") {
		t.Errorf("Expected specific missing asset error, got: %v", err)
	}
}

func TestService_VerifyVersionIntegrity(t *testing.T) {
	env := testenv.New(t)
	actorID := "integrity-tester"

	// 1. Setup
	env.SeedTestUser(t, actorID, "integrity@demo.local", "admin")

	sopID, err := env.SOPService.RegisterSOP("Tamper Detection Protocol", &actorID)
	if err != nil {
		t.Fatalf("Failed to create SOP: %v", err)
	}

	originalContent := "# Highly Secure Data\nDo not change."
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, originalContent, "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to register version: %v", err)
	}

	// 2. Happy Path: Verify the pristine, untouched file
	valid, err := env.SOPService.VerifyVersionIntegrity(v1ID)
	if err != nil {
		t.Fatalf("VerifyVersionIntegrity returned unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("Expected freshly created version to pass integrity check, but it failed")
	}

	// 3. Unhappy Path: Maliciously tamper with the physical file
	path, err := env.SOPService.GetVersionPath(v1ID)
	if err != nil {
		t.Fatalf("Failed to get secure version path: %v", err)
	}

	// We bypass the database entirely and write garbage directly to the hard drive
	err = os.WriteFile(path, []byte("I have hacked your SOP database!"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with file: %v", err)
	}

	// 4. Verify that the tamper detection catches the altered bytes
	valid, err = env.SOPService.VerifyVersionIntegrity(v1ID)
	// Depending on how your internal `integrity` package is written, a hash mismatch
	// might return an error, or it might just return `false, nil`.
	// We just care that `valid` is strictly false!
	if valid {
		t.Fatal("CRITICAL: Tamper detection failed! Integrity check returned true for an altered file.")
	}

	// 5. Unhappy Path: Non-existent Version ID
	_, err = env.SOPService.VerifyVersionIntegrity("fake-uuid-123")
	if err == nil {
		t.Error("Expected an error when verifying a non-existent version ID, got nil")
	}
}
