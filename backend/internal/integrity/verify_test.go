package integrity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile.txt")

	content := []byte("sopandgo content")
	expectedHash := HashBytes(content)

	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid file", func(t *testing.T) {
		valid, err := VerifyFile(tmpFile, expectedHash)
		if err != nil || !valid {
			t.Errorf("Expected true, nil; got %v, %v", valid, err)
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		_, err := VerifyFile("non_existent_file.txt", "anyhash")
		if err == nil {
			t.Error("Expected an error for non-existent file, got nil")
		}
	})

	t.Run("mismatched hash", func(t *testing.T) {
		valid, err := VerifyFile(tmpFile, "wronghash")
		if err != nil || valid {
			t.Errorf("Expected false, nil; got %v, %v", valid, err)
		}
	})
}
