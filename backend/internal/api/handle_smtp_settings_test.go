package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_SMTPSettings(t *testing.T) {
	env := testenv.New(t)
	// testenv saves an SMTP row so email mode can send; this test starts from nothing.
	if _, err := env.Store.DB.Exec(`DELETE FROM smtp_settings`); err != nil {
		t.Fatal(err)
	}

	var fwdSeq atomic.Uint64

	adminID := "smtp-admin-1"
	viewerID := "smtp-viewer-1"
	env.SeedTestUser(t, adminID, "admin@smtp.local", auth.RoleAdmin)
	env.SeedTestUser(t, viewerID, "viewer@smtp.local", auth.RoleViewer)

	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	doRequest := func(method, path, token string, body map[string]any) *httptest.ResponseRecorder {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, path, reqBody)
		// Each request uses a distinct RemoteAddr so this suite does not share one rate-limit bucket.
		n := fwdSeq.Add(1)
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:1234", int((n-1)%200)+1)
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

	t.Run("GET email alias matches smtp path", func(t *testing.T) {
		recSmtp := doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		recEmail := doRequest(http.MethodGet, "/api/admin/settings/email", adminToken, nil)
		if recSmtp.Code != http.StatusOK || recEmail.Code != http.StatusOK {
			t.Fatalf("expected 200 on both paths, got smtp=%d email=%d", recSmtp.Code, recEmail.Code)
		}
		if recSmtp.Body.String() != recEmail.Body.String() {
			t.Fatalf("GET /email body should match GET /smtp")
		}
	})

	t.Run("GET initial", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out["configured"] != false {
			t.Fatalf("expected configured false, got %v", out["configured"])
		}
		if out["mail_mode"] != "smtp" {
			t.Fatalf("expected default mail_mode smtp, got %v", out["mail_mode"])
		}
		if out["mail_transport"] != "smtp" {
			t.Fatalf("expected default mail_transport smtp, got %v", out["mail_transport"])
		}
	})

	t.Run("PUT save", func(t *testing.T) {
		payload := map[string]any{
			"host":         "smtp.example.com",
			"port":         "587",
			"username":     "user",
			"from_address": "no-reply@example.com",
			"password":     "secret-smtp-pass",
		}
		rec := doRequest(http.MethodPut, "/api/admin/settings/smtp", adminToken, payload)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("PATCH mail transport and PUT resend settings", func(t *testing.T) {
		rec := doRequest(http.MethodPatch, "/api/admin/settings/mail-transport", adminToken, map[string]any{
			"mail_transport": "resend",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodPut, "/api/admin/settings/resend", adminToken, map[string]any{
			"from_address": "onboarding@resend.dev",
			"api_key":      "re_test_key_placeholder",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204 resend save, got %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out["mail_transport"] != "resend" {
			t.Fatalf("expected resend transport, got %v", out["mail_transport"])
		}
		if out["resend_configured"] != true {
			t.Fatalf("expected resend_configured true")
		}
		if out["resend_api_key_configured"] != true {
			t.Fatalf("expected resend_api_key_configured true")
		}
	})

	t.Run("GET after save", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var out map[string]any
		json.NewDecoder(rec.Body).Decode(&out)
		if out["host"] != "smtp.example.com" {
			t.Fatalf("host: %v", out["host"])
		}
		if out["password_configured"] != true {
			t.Fatalf("expected password_configured true")
		}
	})

	t.Run("RBAC viewer", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/settings/smtp", viewerToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("PATCH mail mode", func(t *testing.T) {
		payload := map[string]any{
			"mail_mode": "manual_links",
		}
		rec := doRequest(http.MethodPatch, "/api/admin/settings/mail-mode", adminToken, payload)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var out map[string]any
		json.NewDecoder(rec.Body).Decode(&out)
		if out["mail_mode"] != "manual_links" {
			t.Fatalf("expected manual_links, got %v", out["mail_mode"])
		}
	})

	t.Run("PUT smtp does not change mail mode", func(t *testing.T) {
		payload := map[string]any{
			"host":         "smtp.changed.example.com",
			"port":         "587",
			"username":     "user2",
			"from_address": "noreply2@example.com",
			"password":     "",
			"mail_mode":    "smtp", // should be ignored by this endpoint
		}
		rec := doRequest(http.MethodPut, "/api/admin/settings/smtp", adminToken, payload)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/smtp", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var out map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out["mail_mode"] != "manual_links" {
			t.Fatalf("expected mail_mode unchanged (manual_links), got %v", out["mail_mode"])
		}
	})
}
