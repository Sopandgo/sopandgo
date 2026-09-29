package sop_test

import (
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_RegisterSOP(t *testing.T) {
	env := testenv.New(t)

	adminID := "admin-user-1"

	// FIX: Seed the user into the database so the audit log's Foreign Key constraint passes!
	_, err := env.Store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Admin', 'admin@test.local', 'hash', 'admin', 1, '2026-01-01T00:00:00Z')
	`, adminID)
	if err != nil {
		t.Fatalf("Failed to seed admin user: %v", err)
	}

	tests := []struct {
		name        string
		title       string
		actorID     *string
		expectError bool
	}{
		{
			name:        "successfully creates SOP with actor",
			title:       "Emergency Evacuation Procedure",
			actorID:     &adminID,
			expectError: false,
		},
		{
			name:        "successfully creates SOP without actor (system action)",
			title:       "Automated Server Restart",
			actorID:     nil, // This works because NULL is allowed by your FK constraint!
			expectError: false,
		},
		{
			name:        "fails on duplicate title (enforced by DB schema UNIQUE constraint)",
			title:       "Emergency Evacuation Procedure",
			actorID:     &adminID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := env.SOPService.RegisterSOP(tt.title, tt.actorID)

			if (err != nil) != tt.expectError {
				t.Fatalf("RegisterSOP() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if id == "" {
					t.Errorf("Expected valid UUID, got empty string")
				}

				gotSop, err := env.SOPService.GetSOPByID(id)
				if err != nil {
					t.Fatalf("Failed to fetch newly created SOP: %v", err)
				}
				if gotSop.Title != tt.title {
					t.Errorf("Expected title %q, got %q", tt.title, gotSop.Title)
				}
			}
		})
	}
}

func TestService_GetSOPByID(t *testing.T) {
	env := testenv.New(t)
	actorID := "admin-1"

	// FIX: Seed the user into the database first!
	_, err := env.Store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Admin', 'admin@test.local', 'hash', 'admin', 1, '2026-01-01T00:00:00Z')
	`, actorID)
	if err != nil {
		t.Fatalf("Failed to seed admin user: %v", err)
	}

	// Using the address of the variable directly
	id, err := env.SOPService.RegisterSOP("Standard Onboarding", &actorID)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	sop, err := env.SOPService.GetSOPByID(id)
	if err != nil {
		t.Fatalf("Expected to find SOP, got error: %v", err)
	}
	if sop.Title != "Standard Onboarding" {
		t.Errorf("Title mismatch. Got %s", sop.Title)
	}

	_, err = env.SOPService.GetSOPByID("invalid-uuid-1234")
	if err == nil {
		t.Error("Expected an error when fetching a non-existent SOP, got nil")
	}
}

func TestService_ListSOPs(t *testing.T) {
	env := testenv.New(t)
	actorID := "admin-1"

	// Seed the user into the database first!
	_, err := env.Store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Admin', 'admin@test.local', 'hash', 'admin', 1, '2026-01-01T00:00:00Z')
	`, actorID)
	if err != nil {
		t.Fatalf("Failed to seed admin user: %v", err)
	}

	// 1. Empty Database Check
	sops, total, err := env.SOPService.ListSOPs(actorID, 50, 0, "", "", false, false)
	if err != nil {
		t.Fatalf("ListSOPs returned error on empty DB: %v", err)
	}
	if len(sops) != 0 || total != 0 {
		t.Errorf("Expected 0 SOPs and total 0, got len %d, total %d", len(sops), total)
	}

	// Create 3 distinct SOPs
	sopA, _ := env.SOPService.RegisterSOP("Alpha Fire Protocol", &actorID)
	time.Sleep(10 * time.Millisecond)
	_, _ = env.SOPService.RegisterSOP("Beta Water Protocol", &actorID)
	time.Sleep(10 * time.Millisecond)
	sopC, _ := env.SOPService.RegisterSOP("Gamma Earth Protocol", &actorID)

	// 2. Default List (No filters, 50 limit)
	sops, total, err = env.SOPService.ListSOPs(actorID, 50, 0, "", "", false, false)
	if err != nil {
		t.Fatalf("ListSOPs failed: %v", err)
	}
	if len(sops) != 3 || total != 3 {
		t.Fatalf("Expected 3 SOPs, got len %d, total %d", len(sops), total)
	}
	if sops[0].Title != "Gamma Earth Protocol" {
		t.Errorf("Expected newest SOP ('Gamma Earth Protocol') to be first, got %q", sops[0].Title)
	}

	// 3. Pagination Test
	sopsPage1, totalPaginated, _ := env.SOPService.ListSOPs(actorID, 2, 0, "", "", false, false)
	if len(sopsPage1) != 2 || totalPaginated != 3 {
		t.Errorf("Pagination Limit failed. Expected 2 items out of 3 total, got len %d total %d", len(sopsPage1), totalPaginated)
	}

	sopsPage2, _, _ := env.SOPService.ListSOPs(actorID, 2, 2, "", "", false, false)
	if len(sopsPage2) != 1 || sopsPage2[0].Title != "Alpha Fire Protocol" {
		t.Errorf("Pagination Offset failed. Expected 1 item ('Alpha Fire Protocol'), got len %d", len(sopsPage2))
	}

	// 4. Text Search Test (Case Insensitive)
	searchResults, totalSearch, _ := env.SOPService.ListSOPs(actorID, 50, 0, "", "water", false, false)
	if len(searchResults) != 1 || totalSearch != 1 || searchResults[0].Title != "Beta Water Protocol" {
		t.Errorf("Search failed. Expected 1 match for 'water', got %d", len(searchResults))
	}

	// 5. Tag Filtering Test
	tagID, _ := env.SOPService.CreateTag("Elemental", &actorID)
	// Tag SOP A and SOP C, but NOT SOP B
	env.SOPService.AttachTagToSOP(sopA, tagID, &actorID)
	env.SOPService.AttachTagToSOP(sopC, tagID, &actorID)

	tagResults, totalTags, _ := env.SOPService.ListSOPs(actorID, 50, 0, tagID, "", false, false)
	if len(tagResults) != 2 || totalTags != 2 {
		t.Errorf("Tag filter failed. Expected 2 tagged SOPs, got %d", len(tagResults))
	}
}

func TestService_SOPFavorites(t *testing.T) {
	env := testenv.New(t)
	u1 := "fav-user-1"
	u2 := "fav-user-2"
	env.SeedTestUser(t, u1, u1+"@t.local", auth.RoleEditor)
	env.SeedTestUser(t, u2, u2+"@t.local", auth.RoleViewer)

	sop1, err := env.SOPService.RegisterSOP("Alpha Fav", &u1)
	if err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	sop2, err := env.SOPService.RegisterSOP("Beta Fav", &u1)
	if err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}

	list, _, err := env.SOPService.ListSOPs(u1, 50, 0, "", "", false, false)
	if err != nil {
		t.Fatalf("ListSOPs: %v", err)
	}
	for _, s := range list {
		if s.IsFavorite {
			t.Errorf("unexpected favorite before toggle: %s", s.ID)
		}
	}

	if err := env.SOPService.FavoriteSOP(sop1, u1); err != nil {
		t.Fatalf("FavoriteSOP: %v", err)
	}
	if err := env.SOPService.FavoriteSOP(sop1, u1); err != nil {
		t.Fatalf("FavoriteSOP idempotent: %v", err)
	}

	withFav, _, err := env.SOPService.ListSOPs(u1, 50, 0, "", "", false, false)
	if err != nil {
		t.Fatalf("ListSOPs: %v", err)
	}
	var saw1, saw2 bool
	for _, s := range withFav {
		if s.ID == sop1 && !s.IsFavorite {
			t.Error("sop1 should be favorite")
		}
		if s.ID == sop2 && s.IsFavorite {
			t.Error("sop2 should not be favorite")
		}
		if s.ID == sop1 {
			saw1 = true
		}
		if s.ID == sop2 {
			saw2 = true
		}
	}
	if !saw1 || !saw2 {
		t.Fatalf("expected both sops in list")
	}

	only, total, err := env.SOPService.ListSOPs(u1, 50, 0, "", "", true, false)
	if err != nil {
		t.Fatalf("ListSOPs favorites_only: %v", err)
	}
	if total != 1 || len(only) != 1 || only[0].ID != sop1 {
		t.Fatalf("favorites_only: want 1 sop %s, got total=%d len=%d", sop1, total, len(only))
	}

	only2, total2, err := env.SOPService.ListSOPs(u2, 50, 0, "", "", true, false)
	if err != nil {
		t.Fatalf("ListSOPs u2: %v", err)
	}
	if total2 != 0 || len(only2) != 0 {
		t.Fatalf("other user favorites_only should be empty, got total=%d", total2)
	}

	first, _, err := env.SOPService.ListSOPs(u1, 10, 0, "", "", false, true)
	if err != nil {
		t.Fatalf("ListSOPs favorites_first: %v", err)
	}
	if len(first) < 2 || first[0].ID != sop1 {
		t.Fatalf("favorites_first: want sop1 first, got %#v", first[0].ID)
	}

	detail, err := env.SOPService.GetSOPByIDWithTags(sop1, u1)
	if err != nil {
		t.Fatalf("GetSOPByIDWithTags u1: %v", err)
	}
	if !detail.IsFavorite {
		t.Fatal("expected is_favorite for u1")
	}
	detail2, err := env.SOPService.GetSOPByIDWithTags(sop1, u2)
	if err != nil {
		t.Fatalf("GetSOPByIDWithTags u2: %v", err)
	}
	if detail2.IsFavorite {
		t.Fatal("u2 should not see u1 favorite")
	}

	if err := env.SOPService.UnfavoriteSOP(sop1, u1); err != nil {
		t.Fatalf("UnfavoriteSOP: %v", err)
	}
	if err := env.SOPService.UnfavoriteSOP(sop1, u1); err != nil {
		t.Fatalf("UnfavoriteSOP idempotent: %v", err)
	}
	onlyAfter, _, _ := env.SOPService.ListSOPs(u1, 50, 0, "", "", true, false)
	if len(onlyAfter) != 0 {
		t.Fatalf("after unfavorite favorites_only should be empty, got %d", len(onlyAfter))
	}

	if err := env.SOPService.FavoriteSOP("nonexistent-sop-id", u1); err == nil {
		t.Fatal("FavoriteSOP missing sop should error")
	}
}
