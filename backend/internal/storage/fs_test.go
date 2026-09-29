// storage/path_test.go
package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	// Use an absolute base path for testing
	baseDir, _ := filepath.Abs("/var/sopandgo/data")

	tests := []struct {
		name        string
		relPath     string
		expectError bool
		// We only check the suffix because Windows/Linux absolute paths differ (C:\ vs /)
		expectedSuffix string
	}{
		{
			name:           "simple valid path",
			relPath:        "file.txt",
			expectError:    false,
			expectedSuffix: filepath.Join("sopandgo", "data", "file.txt"),
		},
		{
			name:           "nested valid path",
			relPath:        "assets/img/logo.png",
			expectError:    false,
			expectedSuffix: filepath.Join("data", "assets", "img", "logo.png"),
		},
		{
			name:           "messy but valid path",
			relPath:        "assets/../file.txt",
			expectError:    false,
			expectedSuffix: filepath.Join("sopandgo", "data", "file.txt"),
		},
		{
			name:        "malicious directory traversal",
			relPath:     "../../../etc/passwd",
			expectError: true,
		},
		{
			name:    "malicious traversal via absolute path jump",
			relPath: "/etc/passwd",
			// filepath.Join("/base", "/etc") strips the leading slash and joins them.
			// So this becomes /base/etc/passwd, which is SAFE and doesn't escape!
			expectError:    false,
			expectedSuffix: filepath.Join("data", "etc", "passwd"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeJoin(baseDir, tt.relPath)

			if (err != nil) != tt.expectError {
				t.Fatalf("SafeJoin() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError && !strings.HasSuffix(result, tt.expectedSuffix) {
				t.Errorf("SafeJoin() = %v, expected it to end with %v", result, tt.expectedSuffix)
			}
		})
	}
}
