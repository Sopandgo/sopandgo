package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_Tags(t *testing.T) {
	env := testenv.New(t)

	// --- SETUP USERS & TOKENS ---
	editorID := "editor-tag-1"
	adminID := "admin-tag-1"
	viewerID := "viewer-tag-1"

	env.SeedTestUser(t, editorID, "editor@tags.local", auth.RoleEditor)
	env.SeedTestUser(t, adminID, "admin@tags.local", auth.RoleAdmin)
	env.SeedTestUser(t, viewerID, "viewer@tags.local", auth.RoleViewer)

	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	var ipCounter int

	// Helper to send HTTP requests with JSON bodies and spoofed IPs to bypass Rate Limiter
	doJSONRequest := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)

		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	// --- SHARED STATE ---
	var tagID string
	var sopID string

	// 1. Create a base SOP to attach tags to later
	sopID, err := env.SOPService.RegisterSOP("Tagging Test Document", &editorID)
	if err != nil {
		t.Fatalf("Failed to register base SOP: %v", err)
	}

	// =========================================================================
	// TEST CASES
	// =========================================================================

	t.Run("Create Tag (Happy Path)", func(t *testing.T) {
		payload := map[string]string{"title": "Security"}
		rec := doJSONRequest(http.MethodPost, "/api/tags", editorToken, payload)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		tagID = resp["id"]

		if tagID == "" {
			t.Error("Expected tag ID in response, got empty string")
		}
	})

	t.Run("Create Tag (Conflict - 409)", func(t *testing.T) {
		payload := map[string]string{"title": "security"} // Testing case-insensitive conflict
		rec := doJSONRequest(http.MethodPost, "/api/tags", editorToken, payload)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected 409 Conflict for duplicate tag, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("RBAC: Viewer Cannot Create Tag", func(t *testing.T) {
		payload := map[string]string{"title": "HR"}
		rec := doJSONRequest(http.MethodPost, "/api/tags", viewerToken, payload)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for Viewer, got %d", rec.Code)
		}
	})

	t.Run("List Tags", func(t *testing.T) {
		rec := doJSONRequest(http.MethodGet, "/api/tags", viewerToken, nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		var tags []sop.Tag
		json.NewDecoder(rec.Body).Decode(&tags)

		if len(tags) != 1 || tags[0].Title != "Security" {
			t.Errorf("Expected 1 tag named 'Security', got %+v", tags)
		}
	})

	t.Run("Attach Tag to SOP", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/tags/%s", sopID, tagID)
		rec := doJSONRequest(http.MethodPost, path, editorToken, nil)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Detach Tag from SOP", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/tags/%s", sopID, tagID)
		rec := doJSONRequest(http.MethodDelete, path, editorToken, nil)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Set Tag Status (Admin Only)", func(t *testing.T) {
		path := fmt.Sprintf("/api/tags/%s/status", tagID)
		payload := map[string]bool{"is_active": false}

		// 1. Editor should be rejected (403)
		recEditor := doJSONRequest(http.MethodPatch, path, editorToken, payload)
		if recEditor.Code != http.StatusForbidden {
			t.Errorf("Expected Editor to be forbidden from retiring tag, got %d", recEditor.Code)
		}

		// 2. Admin should succeed (200)
		recAdmin := doJSONRequest(http.MethodPatch, path, adminToken, payload)
		if recAdmin.Code != http.StatusOK {
			t.Fatalf("Expected Admin to succeed, got %d: %s", recAdmin.Code, recAdmin.Body.String())
		}
	})

	t.Run("Set Tag Status (Not Found)", func(t *testing.T) {
		path := "/api/tags/fake-uuid-1234/status"
		payload := map[string]bool{"is_active": false}
		rec := doJSONRequest(http.MethodPatch, path, adminToken, payload)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", rec.Code)
		}
	})
}
