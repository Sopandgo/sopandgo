package sop_test

import (
	"os"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_RunSystemIntegrityCheck(t *testing.T) {
	env := testenv.New(t)
	actorID := "admin-1"
	env.SeedTestUser(t, actorID, "user1@demo.local", "admin")

	// ---------------------------------------------------------
	// PHASE 1: SETUP PRISTINE SYSTEM
	// ---------------------------------------------------------
	sopID, _ := env.SOPService.RegisterSOP("System Health Check Protocol", &actorID)

	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Step 1", "Updated procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to register version: %v", err)
	}

	assetID, err := env.SOPService.AddAsset(sopID, "logo.png", []byte("fake png bytes"), &actorID)
	if err != nil {
		t.Fatalf("Failed to add asset: %v", err)
	}

	// 1. Verify Pristine State
	report, err := env.SOPService.RunSystemIntegrityCheck()
	if err != nil {
		t.Fatalf("RunSystemIntegrityCheck failed: %v", err)
	}
	if !report.OK {
		t.Errorf("Expected system to be OK, but it was not")
	}
	if report.Versions.Valid != 1 || report.Versions.Invalid != 0 {
		t.Errorf("Expected 1 valid version and 0 invalid, got %d valid, %d invalid", report.Versions.Valid, report.Versions.Invalid)
	}
	if report.Assets.Valid != 1 || report.Assets.Invalid != 0 {
		t.Errorf("Expected 1 valid asset and 0 invalid, got %d valid, %d invalid", report.Assets.Valid, report.Assets.Invalid)
	}

	// ---------------------------------------------------------
	// PHASE 2: THE TAMPERING EVENT
	// ---------------------------------------------------------
	// We maliciously alter the markdown file directly on the disk
	vPath, _ := env.SOPService.GetVersionPath(v1ID)
	err = os.WriteFile(vPath, []byte("# Step 1 - HACKED"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with version file: %v", err)
	}

	report, err = env.SOPService.RunSystemIntegrityCheck()
	if err != nil {
		t.Fatalf("RunSystemIntegrityCheck failed: %v", err)
	}

	// The system MUST detect the failure
	if report.OK {
		t.Errorf("CRITICAL: System reported OK even though a version was tampered with!")
	}
	if report.Versions.Invalid != 1 {
		t.Errorf("Expected 1 invalid version, got %d", report.Versions.Invalid)
	}
	// The asset should still be perfectly fine
	if report.Assets.Invalid != 0 {
		t.Errorf("Expected 0 invalid assets, got %d", report.Assets.Invalid)
	}
	if len(report.Versions.Failed) != 1 || report.Versions.Failed[0].Version != 1 {
		t.Errorf("Expected failure report to specifically flag Version 1")
	}

	// ---------------------------------------------------------
	// PHASE 3: THE DELETION EVENT
	// ---------------------------------------------------------
	// We simulate an accidental deletion of a file from the server
	aPath, _ := env.SOPService.GetAssetPath(assetID)
	err = os.Remove(aPath)
	if err != nil {
		t.Fatalf("Failed to delete asset file: %v", err)
	}

	report, err = env.SOPService.RunSystemIntegrityCheck()
	if err != nil {
		t.Fatalf("RunSystemIntegrityCheck failed: %v", err)
	}

	// The system should now report BOTH the tampered version and the missing asset
	if report.OK {
		t.Errorf("CRITICAL: System reported OK when it should be failing")
	}
	if report.Versions.Invalid != 1 {
		t.Errorf("Expected 1 invalid version, got %d", report.Versions.Invalid)
	}
	if report.Assets.Invalid != 1 {
		t.Errorf("Expected 1 invalid asset, got %d", report.Assets.Invalid)
	}
	if len(report.Assets.Failed) != 1 || report.Assets.Failed[0].AssetID != assetID {
		t.Errorf("Expected failure report to specifically flag the deleted asset")
	}
}
