package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_AdminEndpoints(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users
	adminID := "super-admin-1"
	viewerID := "basic-viewer-1"
	targetUserID := "target-user-1"

	env.SeedTestUser(t, adminID, "admin@api.local", auth.RoleAdmin)
	env.SeedTestUser(t, viewerID, "viewer@api.local", auth.RoleViewer)
	env.SeedTestUser(t, targetUserID, "target@api.local", auth.RoleEditor)

	// 2. Generate Tokens
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	var ipCounter int

	// Helper to fire HTTP requests with optional JSON body
	doRequest := func(method, path, token string, body map[string]any) *httptest.ResponseRecorder {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}

		req, _ := http.NewRequest(method, path, reqBody)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)
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

	t.Run("System Integrity Check", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/integrity", adminToken, nil)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var report map[string]any
		json.NewDecoder(rec.Body).Decode(&report)
		if ok, exists := report["ok"].(bool); !exists || !ok {
			t.Error("Expected integrity report to return ok=true")
		}
	})

	t.Run("List Users", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/users", adminToken, nil)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var users []map[string]any
		json.NewDecoder(rec.Body).Decode(&users)
		if len(users) < 3 {
			t.Errorf("Expected at least 3 users (seeded), got %d", len(users))
		}
	})

	t.Run("Register New User (Mailer Integration)", func(t *testing.T) {
		payload := map[string]any{
			"display_name": "Brand New Hire",
			"email":        "newhire@api.local",
			"role":         auth.RoleEditor,
		}

		// Reset mail counter before test
		initialMailCount := env.MockMailSender.SentCount

		rec := doRequest(http.MethodPost, "/api/admin/users/register", adminToken, payload)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// PROVE THE EMAIL WAS SENT!
		if env.MockMailSender.SentCount != initialMailCount+1 {
			t.Errorf("Expected MockMailSender to send 1 email, but it didn't!")
		}
	})

	t.Run("Trigger Password Reset (Mailer Integration)", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%s/reset-password", targetUserID)
		initialMailCount := env.MockMailSender.SentCount

		rec := doRequest(http.MethodPost, path, adminToken, nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		if env.MockMailSender.SentCount != initialMailCount+1 {
			t.Errorf("Expected MockMailSender to send 1 email, but it didn't!")
		}
	})

	t.Run("Manual links mode skips mailer", func(t *testing.T) {
		rec := doRequest(http.MethodPatch, "/api/admin/settings/mail-mode", adminToken, map[string]any{
			"mail_mode": "manual_links",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected 204 when switching mode, got %d: %s", rec.Code, rec.Body.String())
		}

		initialMailCount := env.MockMailSender.SentCount
		rec = doRequest(http.MethodPost, "/api/admin/users/register", adminToken, map[string]any{
			"display_name": "Manual Link User",
			"email":        "manual-link@api.local",
			"role":         auth.RoleViewer,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 for register in manual mode, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode register response: %v", err)
		}
		if _, ok := out["invite_link"].(string); !ok {
			t.Fatalf("expected invite_link in response, got: %v", out)
		}

		if env.MockMailSender.SentCount != initialMailCount {
			t.Fatalf("expected no email send in manual mode")
		}

		resetPath := fmt.Sprintf("/api/admin/users/%s/reset-password", targetUserID)
		rec = doRequest(http.MethodPost, resetPath, adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 for reset in manual mode, got %d: %s", rec.Code, rec.Body.String())
		}
		out = map[string]any{}
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode manual reset response: %v", err)
		}
		if _, ok := out["link"].(string); !ok {
			t.Fatalf("expected reset link in manual mode response, got: %v", out)
		}
		if env.MockMailSender.SentCount != initialMailCount {
			t.Fatalf("expected no email send for reset in manual mode")
		}
	})

	t.Run("Email mode without a saved transport falls back to manual links", func(t *testing.T) {
		// Email on, but Resend (never saved) chosen: nothing can send, so invites get a link.
		for path, body := range map[string]map[string]any{
			"/api/admin/settings/mail-mode":      {"mail_mode": "smtp"},
			"/api/admin/settings/mail-transport": {"mail_transport": "resend"},
		} {
			if rec := doRequest(http.MethodPatch, path, adminToken, body); rec.Code != http.StatusNoContent {
				t.Fatalf("PATCH %s: %d %s", path, rec.Code, rec.Body.String())
			}
		}
		t.Cleanup(func() {
			doRequest(http.MethodPatch, "/api/admin/settings/mail-transport", adminToken, map[string]any{"mail_transport": "smtp"})
		})

		initialMailCount := env.MockMailSender.SentCount
		rec := doRequest(http.MethodPost, "/api/admin/users/register", adminToken, map[string]any{
			"display_name": "Fallback Link User",
			"email":        "fallback-link@api.local",
			"role":         auth.RoleViewer,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode register response: %v", err)
		}
		if _, ok := out["invite_link"].(string); !ok {
			t.Fatalf("expected invite_link from the fallback, got: %v", out)
		}
		if env.MockMailSender.SentCount != initialMailCount {
			t.Fatal("expected no email without a saved transport")
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/email", adminToken, nil)
		var pub map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&pub)
		if pub["mail_mode"] != "smtp" || pub["effective_mail_mode"] != "manual_links" {
			t.Fatalf("expected mail_mode smtp with effective manual_links, got %v / %v", pub["mail_mode"], pub["effective_mail_mode"])
		}
	})

	t.Run("SMTP mode failure does not return fallback link", func(t *testing.T) {
		rec := doRequest(http.MethodPatch, "/api/admin/settings/mail-mode", adminToken, map[string]any{
			"mail_mode": "smtp",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("Expected 204 when switching back to smtp, got %d: %s", rec.Code, rec.Body.String())
		}

		env.MockMailSender.SetFail(true)
		defer env.MockMailSender.SetFail(false)

		rec = doRequest(http.MethodPost, "/api/admin/users/register", adminToken, map[string]any{
			"display_name": "SMTP Fail User",
			"email":        "smtp-fail@api.local",
			"role":         auth.RoleViewer,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 on register with smtp failure, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode smtp-fail register response: %v", err)
		}
		if _, hasLink := out["link"]; hasLink {
			t.Fatalf("did not expect fallback invite link in smtp mode response: %v", out)
		}

		rec = doRequest(http.MethodPost, fmt.Sprintf("/api/admin/users/%s/reset-password", targetUserID), adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 on reset with smtp failure, got %d: %s", rec.Code, rec.Body.String())
		}
		out = map[string]any{}
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatalf("decode smtp-fail reset response: %v", err)
		}
		if _, hasLink := out["link"]; hasLink {
			t.Fatalf("did not expect fallback reset link in smtp mode response: %v", out)
		}
	})

	t.Run("Update User Role", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%s/update-role", targetUserID)
		payload := map[string]any{"new_role": auth.RoleViewer}

		rec := doRequest(http.MethodPatch, path, adminToken, payload)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d", rec.Code)
		}

		// Verify DB change
		role, _ := env.AuthService.GetUserRole(targetUserID)
		if role != auth.RoleViewer {
			t.Errorf("Expected role to be updated to Viewer, got %s", role)
		}
	})

	t.Run("Update User Status (Self-Disable Protection)", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%s/update-status", adminID)
		payload := map[string]any{"active": false}

		rec := doRequest(http.MethodPatch, path, adminToken, payload)

		// Your handler specifically maps ErrSelfDisable to 403 Forbidden!
		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for self-disable, got %d", rec.Code)
		}
	})

	t.Run("Update User Status (Target User)", func(t *testing.T) {
		path := fmt.Sprintf("/api/admin/users/%s/update-status", targetUserID)
		payload := map[string]any{"active": false}

		rec := doRequest(http.MethodPatch, path, adminToken, payload)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d", rec.Code)
		}

		// Verify DB change
		isActive, _ := env.AuthService.GetUserActiveStatus(targetUserID)
		if isActive {
			t.Error("Expected target user to be inactive, but was still active")
		}
	})

	t.Run("RBAC Rejection: Viewer accessing Admin endpoints", func(t *testing.T) {
		// Try to list users using a basic Viewer token
		rec := doRequest(http.MethodGet, "/api/admin/users", viewerToken, nil)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for Viewer accessing admin route, got %d", rec.Code)
		}
	})
}
