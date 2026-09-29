package sop_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_AssetLifecycle(t *testing.T) {
	env := testenv.New(t)
	actorID := "user-1"

	// FIX: Seed the user into the database!
	env.SeedTestUser(t, actorID, "user1@demo.local", "admin")

	// 1. Setup: Create a base SOP container
	sopID, err := env.SOPService.RegisterSOP("Asset Lifecycle SOP", &actorID)
	if err != nil {
		t.Fatalf("Failed to setup SOP: %v", err)
	}

	// 2. Add an Asset (Happy Path)
	filename := "diagram.png"
	content := []byte("fake image payload data")
	assetID, err := env.SOPService.AddAsset(sopID, filename, content, &actorID)

	if err != nil {
		t.Fatalf("AddAsset failed: %v", err)
	}
	if assetID == "" {
		t.Fatal("Expected valid asset ID, got empty string")
	}

	// 3. Prevent Overwrites (Idempotency / Conflict logic)
	// Trying to upload another asset with the same filename to the same SOP must fail.
	_, err = env.SOPService.AddAsset(sopID, filename, []byte("different content"), &actorID)
	if err == nil || !strings.Contains(err.Error(), "asset already exists") {
		t.Fatalf("Expected AddAsset to fail on duplicate filename, got: %v", err)
	}

	// 4. Fetch Asset Metadata
	asset, err := env.SOPService.GetAssetByID(assetID)
	if err != nil {
		t.Fatalf("GetAssetByID failed: %v", err)
	}
	if asset.Filename != filename {
		t.Errorf("Expected filename %q, got %q", filename, asset.Filename)
	}

	// 5. Fetch Asset Content (Verify disk I/O)
	fetchedContent, err := env.SOPService.GetAssetContent(assetID)
	if err != nil {
		t.Fatalf("GetAssetContent failed: %v", err)
	}
	if !bytes.Equal(content, fetchedContent) {
		t.Errorf("Fetched content does not match uploaded content")
	}

	// 6. List Assets for SOP
	assets, err := env.SOPService.GetAssetsBySOPID(sopID)
	if err != nil {
		t.Fatalf("GetAssetsBySOPID failed: %v", err)
	}
	if len(assets) != 1 {
		t.Errorf("Expected exactly 1 asset, got %d", len(assets))
	}
	if assets[0].ID != assetID {
		t.Errorf("Expected asset ID %q, got %q", assetID, assets[0].ID)
	}
}

func TestService_VerifyAssetIntegrity(t *testing.T) {
	env := testenv.New(t)
	actorID := "user-1"

	// FIX: Seed the user into the database!
	env.SeedTestUser(t, actorID, "admin@test.local", "admin")

	// 1. Setup SOP and Asset
	sopID, _ := env.SOPService.RegisterSOP("Integrity Test SOP", &actorID)
	originalContent := []byte("important secure document")
	assetID, err := env.SOPService.AddAsset(sopID, "doc.pdf", originalContent, &actorID)
	if err != nil {
		t.Fatalf("Failed to setup Asset: %v", err) // <-- Good practice to catch this error!
	}

	// 2. Happy Path: Verify Intact File
	valid, err := env.SOPService.VerifyAssetIntegrity(assetID)
	if err != nil {
		t.Fatalf("VerifyAssetIntegrity returned unexpected error: %v", err)
	}
	if !valid {
		t.Fatal("Expected asset integrity to be valid, got invalid")
	}

	// 3. Unhappy Path: Maliciously tamper with the file on disk
	path, err := env.SOPService.GetAssetPath(assetID)
	if err != nil {
		t.Fatalf("Failed to get secure asset path: %v", err)
	}

	// Overwrite the file bytes directly, bypassing the database entirely
	err = os.WriteFile(path, []byte("hacked document payload"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with file: %v", err)
	}

	// 4. Verify Tamper Detection catches the hash mismatch
	valid, err = env.SOPService.VerifyAssetIntegrity(assetID)
	// Some hash functions return an error on mismatch, others return false. We just check valid == false.
	if valid {
		t.Fatal("Tamper detection failed! Integrity check returned true for an altered file")
	}
}
