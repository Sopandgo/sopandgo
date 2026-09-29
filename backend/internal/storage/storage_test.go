// storage/storage_test.go
package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenInMemory_AndMigrations(t *testing.T) {
	// 1. Test Initial Boot
	store, err := OpenInMemory()
	if err != nil {
		t.Fatalf("Failed to initialize in-memory database: %v", err)
	}
	defer store.DB.Close()

	// 2. Verify Schema Version
	var version int
	err = store.DB.QueryRow(`SELECT version FROM schema_version`).Scan(&version)
	if err != nil {
		t.Fatalf("Failed to query schema_version: %v", err)
	}

	expectedVersion := migrations[len(migrations)-1].version
	if version != expectedVersion {
		t.Errorf("Expected database to be at version %d, but got %d", expectedVersion, version)
	}

	// 3. Verify Idempotency (Running migrations again should be perfectly safe and do nothing)
	newVersion, err := runMigrations(store.DB)
	if err != nil {
		t.Fatalf("Running migrations a second time returned an error: %v", err)
	}
	if newVersion != expectedVersion {
		t.Errorf("Expected version to remain %d, got %d", expectedVersion, newVersion)
	}
}

func TestOpen_CreatesFileOnDisk(t *testing.T) {
	// 1. Get a safe, temporary, auto-deleting folder
	tempDir := t.TempDir()

	// 2. Run the production Open function
	store, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer store.DB.Close()

	// 3. Verify the file actually exists on the hard drive
	expectedPath := filepath.Join(tempDir, "app.db")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected physical database file to be created at %s, but it was not found", expectedPath)
	}
}
