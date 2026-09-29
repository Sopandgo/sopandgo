package sop_test

import (
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_TagLifecycle(t *testing.T) {
	env := testenv.New(t)
	actorID := "admin-tag-1"

	// Seed the user into the database so the audit log's Foreign Key constraint passes
	_, err := env.Store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Admin', 'admin@tags.local', 'hash', 'admin', 1, '2026-01-01T00:00:00Z')
	`, actorID)
	if err != nil {
		t.Fatalf("Failed to seed admin user: %v", err)
	}

	t.Run("Create and List Tags", func(t *testing.T) {
		tag1ID, err := env.SOPService.CreateTag("Safety", &actorID)
		if err != nil || tag1ID == "" {
			t.Fatalf("Failed to create tag: %v", err)
		}

		// Test trimming whitespace
		_, err = env.SOPService.CreateTag("  HR  ", &actorID)
		if err != nil {
			t.Fatalf("Failed to create tag with whitespace: %v", err)
		}

		tags, err := env.SOPService.ListTags()
		if err != nil {
			t.Fatalf("Failed to list tags: %v", err)
		}

		if len(tags) != 2 {
			t.Fatalf("Expected 2 tags, got %d", len(tags))
		}

		// Ensure tags are returned sorted alphabetically (HR comes before Safety)
		if tags[0].Title != "HR" || tags[1].Title != "Safety" {
			t.Errorf("Tags are not sorted correctly or whitespace was not trimmed. Got: %s, %s", tags[0].Title, tags[1].Title)
		}
	})

	t.Run("Creation Edge Cases (Empty and Duplicates)", func(t *testing.T) {
		// Empty tag
		_, err := env.SOPService.CreateTag("   ", &actorID)
		if err == nil {
			t.Error("Expected error when creating empty tag, got nil")
		}

		// Exact duplicate
		_, err = env.SOPService.CreateTag("Safety", &actorID)
		if err == nil {
			t.Error("Expected error when creating duplicate tag 'Safety', got nil")
		}

		// Case-insensitive duplicate
		_, err = env.SOPService.CreateTag(" sAfEtY ", &actorID)
		if err == nil {
			t.Error("Expected error when creating case-insensitive duplicate tag ' sAfEtY ', got nil")
		}
	})

	t.Run("Retire and Revive Tags", func(t *testing.T) {
		// Grab a valid tag ID
		tags, _ := env.SOPService.ListTags()
		targetTag := tags[0]

		// Retire it
		err := env.SOPService.SetTagStatus(targetTag.ID, false, &actorID)
		if err != nil {
			t.Fatalf("Failed to retire tag: %v", err)
		}

		// Verify it is inactive
		tags, _ = env.SOPService.ListTags()
		for _, tag := range tags {
			if tag.ID == targetTag.ID && tag.IsActive != false {
				t.Errorf("Expected tag %s to be inactive, but IsActive is true", tag.Title)
			}
		}

		// Revive it
		err = env.SOPService.SetTagStatus(targetTag.ID, true, &actorID)
		if err != nil {
			t.Fatalf("Failed to revive tag: %v", err)
		}
	})
}

func TestService_SOPTagAttachment(t *testing.T) {
	env := testenv.New(t)
	actorID := "admin-tag-2"

	// Seed the user
	env.Store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Admin', 'admin2@tags.local', 'hash', 'admin', 1, '2026-01-01T00:00:00Z')
	`, actorID)

	// 1. Create an SOP
	sopID, err := env.SOPService.RegisterSOP("Tagged Procedure", &actorID)
	if err != nil {
		t.Fatalf("Failed to create SOP: %v", err)
	}

	// 2. Create a Tag
	tagID, err := env.SOPService.CreateTag("Critical", &actorID)
	if err != nil {
		t.Fatalf("Failed to create Tag: %v", err)
	}

	t.Run("Attach Tag", func(t *testing.T) {
		err := env.SOPService.AttachTagToSOP(sopID, tagID, &actorID)
		if err != nil {
			t.Fatalf("Failed to attach tag to SOP: %v", err)
		}

		// Try attaching it again (Should succeed silently due to ON CONFLICT DO NOTHING)
		err = env.SOPService.AttachTagToSOP(sopID, tagID, &actorID)
		if err != nil {
			t.Fatalf("Expected duplicate attachment to succeed silently, got error: %v", err)
		}
	})

	t.Run("Detach Tag", func(t *testing.T) {
		err := env.SOPService.DetachTagFromSOP(sopID, tagID, &actorID)
		if err != nil {
			t.Fatalf("Failed to detach tag from SOP: %v", err)
		}
	})
}
