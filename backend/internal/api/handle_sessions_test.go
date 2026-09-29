package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_AdminSessions(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users
	adminID := "admin-session-1"
	viewerID := "viewer-session-1"
	targetUserID := "target-session-1"

	env.SeedTestUser(t, adminID, "admin@sessions.local", auth.RoleAdmin)
	env.SeedTestUser(t, viewerID, "viewer@sessions.local", auth.RoleViewer)
	env.SeedTestUser(t, targetUserID, "target@sessions.local", auth.RoleEditor)

	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	// 2. Setup Active Sessions in the DB
	// We manually create refresh tokens so they show up in the List query
	env.AuthService.CreateSession(adminID)
	targetToken, _ := env.AuthService.CreateSession(targetUserID)
	env.AuthService.CreateSession(targetUserID) // Target has 2 active devices (e.g., phone & laptop)

	// Helper to fire HTTP requests
	doRequest := func(method, path, token string, body map[string]any) *httptest.ResponseRecorder {
		var reqBody *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		} else {
			reqBody = bytes.NewReader([]byte{})
		}

		req, _ := http.NewRequest(method, path, reqBody)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	// --- RUN TEST CASES ---

	t.Run("RBAC Rejection: Viewer accessing Sessions", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/sessions", viewerToken, nil)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for Viewer, got %d", rec.Code)
		}
	})

	t.Run("List Active Sessions", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/sessions", adminToken, nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		var sessions []map[string]any
		json.NewDecoder(rec.Body).Decode(&sessions)

		// We seeded 3 sessions (1 for admin, 2 for target)
		if len(sessions) != 3 {
			t.Errorf("Expected exactly 3 active sessions, got %d", len(sessions))
		}
		if bytes.Contains(rec.Body.Bytes(), []byte(targetToken)) {
			t.Error("session list returned a live refresh token")
		}
		for _, session := range sessions {
			if _, ok := session["id"]; ok {
				t.Errorf("session list included an id field: %#v", session["id"])
			}
		}
	})

	t.Run("Revoke Specific User's Sessions", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%s/sessions", targetUserID)
		rec := doRequest(http.MethodDelete, path, adminToken, nil)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d", rec.Code)
		}

		// Verify the target's session is actually dead by trying to validate it
		if _, _, err := env.AuthService.ValidateSession(targetToken); err == nil {
			t.Error("CRITICAL: Target's session survived the revocation!")
		}
	})

	t.Run("Revoke All Sessions (Missing Reason)", func(t *testing.T) {
		// Attempting the global kill switch without providing a mandatory reason
		rec := doRequest(http.MethodDelete, "/api/admin/sessions", adminToken, map[string]any{
			"reason": "   ", // Blank or empty string
		})

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for missing reason, got %d", rec.Code)
		}
	})

	t.Run("Revoke All Sessions (Global Kill Switch)", func(t *testing.T) {
		// Valid global kill switch
		payload := map[string]any{"reason": "Suspected database breach"}
		rec := doRequest(http.MethodDelete, "/api/admin/sessions", adminToken, payload)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d", rec.Code)
		}

		// Verify by checking the total list of active sessions (should now be 0)
		listRec := doRequest(http.MethodGet, "/api/admin/sessions", adminToken, nil)
		var sessions []map[string]any
		json.NewDecoder(listRec.Body).Decode(&sessions)

		if len(sessions) != 0 {
			t.Errorf("Expected 0 active sessions after kill switch, got %d", len(sessions))
		}
	})
}
