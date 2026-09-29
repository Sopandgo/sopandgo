package sop_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_AddAcknowledgment(t *testing.T) {
	env := testenv.New(t)
	authorID := "author-100"
	readerID := "reader-100"

	env.SeedTestUser(t, authorID, "admin@test.local", "admin")
	env.SeedTestUser(t, readerID, "reader100@demo.local", "viewer")

	// 1. Setup
	sopID, _ := env.SOPService.RegisterSOP("Ack Test SOP", &authorID)
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "Content", "Updated procedure", &authorID)
	if err != nil {
		t.Fatalf("Failed to register version: %v", err)
	}

	// Note: RegisterSOPVersion automatically creates an 'author' ack.
	// Reader sign-off is allowed only after the version is published.

	_, err = env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeRead)
	if err == nil || !strings.Contains(err.Error(), "conflict:") {
		t.Fatalf("Expected unpublished reader ack to be rejected, got: %v", err)
	}

	if err := env.SOPService.TransitionVersionState(v1ID, sop.StateRC, authorID); err != nil {
		t.Fatalf("Failed to promote version: %v", err)
	}
	if _, err := env.SOPService.ApproveSOPVersion(v1ID, authorID); err != nil {
		t.Fatalf("Failed to publish version: %v", err)
	}

	_, err = env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeApproved)
	if err == nil || !strings.Contains(err.Error(), "conflict:") {
		t.Fatalf("Expected direct approver ack to be rejected, got: %v", err)
	}

	// 2. Add Acknowledgment (Happy Path)
	ackID, err := env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeRead)
	if err != nil {
		t.Fatalf("AddAcknowledgment failed: %v", err)
	}
	if ackID == "" {
		t.Fatal("Expected valid ack ID, got empty string")
	}

	// 3. Test UNIQUE Constraint
	// A user cannot acknowledge the exact same version with the exact same type twice.
	_, err = env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeRead)
	if err == nil {
		t.Fatal("Expected AddAcknowledgment to fail on duplicate user/type/version combination")
	}
	if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
		t.Errorf("Expected UNIQUE constraint error, got: %v", err)
	}
}

func TestService_GetAcknowledgments(t *testing.T) {
	env := testenv.New(t)

	authorID := "author-200"
	readerID := "reader-200"
	env.SeedTestUser(t, authorID, "admin@test.local", "admin")
	env.SeedTestUser(t, readerID, "reader200@demo.local", "viewer")

	// 1. Setup SOP and Version
	sopID, _ := env.SOPService.RegisterSOP("Multi-Ack SOP", &authorID)
	v1ID, _, _ := env.SOPService.RegisterSOPVersion(sopID, "Content", "Updated procedure", &authorID)

	time.Sleep(15 * time.Millisecond)

	// Transition to Published so the reader can "read" it
	env.SOPService.TransitionVersionState(v1ID, sop.StateRC, authorID)
	time.Sleep(15 * time.Millisecond)

	_, err := env.SOPService.ApproveSOPVersion(v1ID, authorID)
	if err != nil {
		t.Fatalf("Failed to approve version: %v", err)
	}

	time.Sleep(15 * time.Millisecond)

	// Add Reader Ack
	_, err = env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeRead)
	if err != nil {
		t.Fatalf("Failed to add reader ack: %v", err)
	}

	// 2. Test GetAcknowledgmentsByVersion
	versionAcks, err := env.SOPService.GetAcknowledgmentsByVersion(v1ID)
	if err != nil {
		t.Fatalf("GetAcknowledgmentsByVersion failed: %v", err)
	}
	// We expect 3: Author (from creation), Approver (from ApproveSOPVersion), Reader (manual)
	if len(versionAcks) != 3 {
		t.Errorf("Expected 3 acks for version, got %d", len(versionAcks))
	}

	// 3. Test GetAcknowledgmentsByUser
	readerAcks, err := env.SOPService.GetAcknowledgmentsByUser(readerID)
	if err != nil {
		t.Fatalf("GetAcknowledgmentsByUser failed: %v", err)
	}
	if len(readerAcks) != 1 {
		t.Errorf("Expected 1 ack for reader, got %d", len(readerAcks))
	}
	if readerAcks[0].AcknowledgmentType != sop.AckTypeRead {
		t.Errorf("Expected ack type 'reader', got %q", readerAcks[0].AcknowledgmentType)
	}

	// 4. Test GetAcknowledgmentsByVersionWithUser (JOIN Query)
	detailedAcks, err := env.SOPService.GetAcknowledgmentsByVersionWithUser(v1ID)
	if err != nil {
		t.Fatalf("GetAcknowledgmentsByVersionWithUser failed: %v", err)
	}
	if len(detailedAcks) != 3 {
		t.Errorf("Expected 3 detailed acks, got %d", len(detailedAcks))
	}

	// Verify the SQL JOIN actually mapped the user email into the nested struct correctly
	foundReaderEmail := false
	for _, ack := range detailedAcks {
		if ack.UserID == readerID {
			foundReaderEmail = true
			if ack.User.Email != "reader200@demo.local" {
				t.Errorf("SQL JOIN failed: Expected email 'reader200@demo.local', got %q", ack.User.Email)
			}
		}
	}
	if !foundReaderEmail {
		t.Error("Did not find the reader's detailed acknowledgment in the JOIN results")
	}
}

func TestService_GetSignatureStatusByUser(t *testing.T) {
	env := testenv.New(t)

	authorID := "author-status"
	readerID := "reader-status"
	env.SeedTestUser(t, authorID, "admin@test.local", "admin")
	env.SeedTestUser(t, readerID, "reader@status.local", "viewer")

	// 1. Setup SOP and Version
	sopID, _ := env.SOPService.RegisterSOP("Status SOP", &authorID)
	v1ID, _, _ := env.SOPService.RegisterSOPVersion(sopID, "V1 Content", "Updated procedure", &authorID)

	time.Sleep(15 * time.Millisecond)

	// Publish V1
	env.SOPService.TransitionVersionState(v1ID, sop.StateRC, authorID)
	_, err := env.SOPService.ApproveSOPVersion(v1ID, authorID)
	if err != nil {
		t.Fatalf("Approve V1 failed: %v", err)
	}
	time.Sleep(15 * time.Millisecond)

	// Check status before reader signs
	statuses, err := env.SOPService.GetSignatureStatusByUser(readerID)
	if err != nil {
		t.Fatalf("GetSignatureStatusByUser failed: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}
	if statuses[0].HasSignedLatest {
		t.Error("Expected HasSignedLatest to be false")
	}
	if statuses[0].SignedOlderVersion {
		t.Error("Expected SignedOlderVersion to be false")
	}

	// Reader signs V1
	_, err = env.SOPService.AddAcknowledgment(v1ID, readerID, sop.AckTypeRead)
	if err != nil {
		t.Fatalf("Failed to add reader ack: %v", err)
	}

	// Check status after reader signs V1
	statuses, _ = env.SOPService.GetSignatureStatusByUser(readerID)
	if !statuses[0].HasSignedLatest {
		t.Error("Expected HasSignedLatest to be true after signing V1")
	}

	// Create and Publish V2
	v2ID, _, _ := env.SOPService.RegisterSOPVersion(sopID, "V2 Content", "Updated procedure", &authorID)
	env.SOPService.TransitionVersionState(v2ID, sop.StateRC, authorID)
	if _, err := env.SOPService.ApproveSOPVersion(v2ID, authorID); err != nil {
		t.Fatalf("Approve V2 failed: %v", err)
	}

	// Check status after V2 published (reader has not signed V2)
	statuses, _ = env.SOPService.GetSignatureStatusByUser(readerID)
	if statuses[0].HasSignedLatest {
		t.Error("Expected HasSignedLatest to be false after V2 published")
	}
	if !statuses[0].SignedOlderVersion {
		t.Error("Expected SignedOlderVersion to be true after V2 published")
	}
}
