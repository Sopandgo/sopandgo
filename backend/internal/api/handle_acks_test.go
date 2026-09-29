package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_Acknowledgments(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users
	// We need multiple users to test different RBAC permission scopes
	author1 := "author-1" // Original creator
	approver := "approver-1"
	reader := "reader-1"
	auditor := "auditor-1"

	env.SeedTestUser(t, author1, "author1@api.local", auth.RoleEditor)
	env.SeedTestUser(t, approver, "approver@api.local", auth.RoleApprover)
	env.SeedTestUser(t, reader, "reader@api.local", auth.RoleViewer)
	env.SeedTestUser(t, auditor, "auditor@api.local", auth.RoleAuditor)

	// 2. Generate Authentication Tokens
	approverToken, _ := auth.GenerateAccessToken(approver, auth.RoleApprover)
	readerToken, _ := auth.GenerateAccessToken(reader, auth.RoleViewer)
	auditorToken, _ := auth.GenerateAccessToken(auditor, auth.RoleAuditor)

	// 3. Setup Database State (Bypassing HTTP for speed)
	sopID, _ := env.SOPService.RegisterSOP("API Target SOP", &author1)
	v1ID, _, err := env.SOPService.RegisterSOPVersion(sopID, "Base Content", "Updated procedure", &author1)
	if err != nil {
		t.Fatalf("Failed to setup DB state: %v", err)
	}

	var ipCounter int

	// Helper to fire HTTP requests directly into the API Router
	doRequest := func(method, path, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, nil)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req) // Hits the actual HTTP multiplexer
		return rec
	}

	// --- RUN TEST CASES ---

	t.Run("Reader Acknowledgment Rejected Until Published", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/add-reader", sopID, v1ID)
		rec := doRequest(http.MethodPost, path, readerToken)

		if rec.Code != http.StatusConflict {
			t.Errorf("Expected 409 Conflict for a draft, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	if err := env.SOPService.TransitionVersionState(v1ID, sop.StateRC, author1); err != nil {
		t.Fatalf("Failed to promote version: %v", err)
	}
	if _, err := env.SOPService.ApproveSOPVersion(v1ID, approver); err != nil {
		t.Fatalf("Failed to publish version: %v", err)
	}

	t.Run("Removed Author and Approver Endpoints", func(t *testing.T) {
		authorPath := fmt.Sprintf("/api/sops/%s/versions/%s/add-author", sopID, v1ID)
		approverPath := fmt.Sprintf("/api/sops/%s/versions/%s/add-approver", sopID, v1ID)
		for _, path := range []string{authorPath, approverPath} {
			rec := doRequest(http.MethodPost, path, approverToken)
			if rec.Code != http.StatusNotFound {
				t.Errorf("Expected 404 for %s, got %d", path, rec.Code)
			}
		}
	})

	t.Run("Add Reader Acknowledgment", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/add-reader", sopID, v1ID)
		rec := doRequest(http.MethodPost, path, readerToken)

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("RBAC Rejection: Auditor cannot reader-sign", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/add-reader", sopID, v1ID)
		rec := doRequest(http.MethodPost, path, auditorToken)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for auditor, got %d", rec.Code)
		}
	})

	t.Run("List Acknowledgments", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/acks", sopID, v1ID)
		rec := doRequest(http.MethodGet, path, readerToken)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Decode the JSON array to ensure the data is coming through
		var acks []map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&acks); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		// Author (from create), approver (from publish), reader (from add-reader).
		if len(acks) != 3 {
			t.Errorf("Expected 3 acknowledgments in JSON response, got %d", len(acks))
		}
	})

	t.Run("Missing Auth Token", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/acks", sopID, v1ID)
		rec := doRequest(http.MethodGet, path, "") // No token provided

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Service Error: Duplicate Acknowledgment (500)", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/%s/add-reader", sopID, v1ID)

		// The first request succeeds (we already did this in an earlier test,
		// but let's do it again to guarantee the duplicate DB constraint triggers)
		doRequest(http.MethodPost, path, readerToken)

		// The second request MUST fail because of SQLite UNIQUE constraints
		rec := doRequest(http.MethodPost, path, readerToken)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 Internal Server Error for duplicate ack, got %d", rec.Code)
		}
	})

	t.Run("Service Error: Invalid Version ID (404)", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/versions/fake-version-uuid/add-reader", sopID)
		rec := doRequest(http.MethodPost, path, readerToken)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found for non-existent version, got %d", rec.Code)
		}
	})

	t.Run("List Errors: Invalid Version ID (500)", func(t *testing.T) {
		// Passing a fake ID should cause the DB lookup to fail or return nothing.
		// If your service returns an error for "not found", it triggers a 500 here.
		path := fmt.Sprintf("/api/sops/%s/versions/fake-version-uuid/acks", sopID)
		rec := doRequest(http.MethodGet, path, readerToken)

		// Note: Depending on how your SQL handles empty joins, this might return 200 with an empty array `[]`.
		// If it returns 500, we catch it here.
		if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusOK {
			t.Errorf("Expected 500 or 200, got %d", rec.Code)
		}
	})
}
