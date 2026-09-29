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
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_AdminIntegrations(t *testing.T) {
	env := testenv.New(t)

	adminID := "integ-admin-1"
	env.SeedTestUser(t, adminID, "integ-admin@api.local", auth.RoleAdmin)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)

	var ipCounter int
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

	t.Run("GET integrations", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/settings/integrations", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
		}
		var pub notify.PublicSettings
		if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
			t.Fatal(err)
		}
		if !pub.EncryptionKeySet {
			t.Fatal("expected encryption_key_set")
		}
	})

	t.Run("PUT slack and gotify and webhook", func(t *testing.T) {
		rec := doRequest(http.MethodPut, "/api/admin/settings/integrations/slack", adminToken, map[string]any{
			"enabled":     true,
			"webhook_url": "https://hooks.slack.com/services/T00/B00/xxx",
			"events":      []string{notify.EventSOPPublished},
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("slack: %d %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodPut, "/api/admin/settings/integrations/gotify", adminToken, map[string]any{
			"enabled": true,
			"url":     "https://gotify.example.com",
			"token":   "app-token",
			"events":  notify.KnownEvents,
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("gotify: %d %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodPut, "/api/admin/settings/integrations/webhook", adminToken, map[string]any{
			"enabled":      true,
			"url":          "https://example.com/hook",
			"bearer_token": "secret",
			"events":       []string{notify.EventSOPPublished, notify.EventBackupS3Failed},
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("webhook: %d %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/integrations", adminToken, nil)
		var pub notify.PublicSettings
		if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
			t.Fatal(err)
		}
		if !pub.Slack.Enabled || !pub.Slack.SecretConfigured {
			t.Fatalf("slack: %#v", pub.Slack)
		}
		if !pub.Gotify.Configured || pub.Gotify.URL != "https://gotify.example.com" {
			t.Fatalf("gotify: %#v", pub.Gotify)
		}
		if !pub.Webhook.Configured || !pub.Webhook.SecretConfigured {
			t.Fatalf("webhook: %#v", pub.Webhook)
		}
	})
}
