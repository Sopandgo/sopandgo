package sop_test

import (
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_ValidateSOPContentAssets(t *testing.T) {
	env := testenv.New(t)
	actorID := "validator-admin"

	// 1. Setup
	env.SeedTestUser(t, actorID, "admin@env.local", "admin")

	sopID, err := env.SOPService.RegisterSOP("Validation Protocol", &actorID)
	if err != nil {
		t.Fatalf("Failed to create SOP: %v", err)
	}

	// Add an existing asset to the database for this SOP
	_, err = env.SOPService.AddAsset(sopID, "diagram.png", []byte("image bytes"), &actorID)
	if err != nil {
		t.Fatalf("Failed to add asset: %v", err)
	}

	// 2. Define Test Cases
	tests := []struct {
		name        string
		content     string
		expectError bool
		errContains string
	}{
		{
			name:        "no assets referenced in markdown",
			content:     "# Standard Text\nNo images here, just words.",
			expectError: false,
		},
		{
			name:        "valid existing asset referenced",
			content:     "Look at this graph: ![My Diagram](assets/diagram.png)",
			expectError: false,
		},
		{
			name:        "missing asset referenced",
			content:     "Missing image: ![Oops](assets/missing-file.png)",
			expectError: true,
			errContains: "referenced asset not found: assets/missing-file.png",
		},
		{
			name:        "multiple assets (one valid, one missing)",
			content:     "Valid: ![v](assets/diagram.png)\nInvalid: ![i](assets/typo.png)",
			expectError: true,
			errContains: "referenced asset not found",
		},
		{
			name:        "ignores external links",
			content:     "External: ![ext](https://google.com/image.png)",
			expectError: false, // Regex specifically looks for (assets/...)
		},
	}

	// 3. Run Test Cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.SOPService.ValidateSOPContentAssets(sopID, tt.content)

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}
