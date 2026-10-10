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
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_AdminBackupS3Settings(t *testing.T) {
	env := testenv.New(t)

	adminID := "s3-admin-1"
	env.SeedTestUser(t, adminID, "s3-admin@api.local", auth.RoleAdmin)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerID := "s3-viewer-1"
	env.SeedTestUser(t, viewerID, "s3-viewer@api.local", auth.RoleViewer)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

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

	settings := map[string]any{
		"enabled":           true,
		"bucket":            "lab-backups",
		"region":            "eu-central-1",
		"key_prefix":        "lab",
		"endpoint":          "http://127.0.0.1:1",
		"access_key_id":     "AKIDEXAMPLE",
		"secret_access_key": "secret-1",
		"interval":          "12h",
		"retention_max":     7,
		"retention_days":    14,
	}

	t.Run("viewers cannot read or change settings", func(t *testing.T) {
		if rec := doRequest(http.MethodGet, "/api/admin/settings/backup-s3", viewerToken, nil); rec.Code != http.StatusForbidden {
			t.Fatalf("GET: expected 403, got %d", rec.Code)
		}
		if rec := doRequest(http.MethodPut, "/api/admin/settings/backup-s3", viewerToken, settings); rec.Code != http.StatusForbidden {
			t.Fatalf("PUT: expected 403, got %d", rec.Code)
		}
	})

	t.Run("run now while off", func(t *testing.T) {
		rec := doRequest(http.MethodPost, "/api/admin/backups/s3/run", adminToken, nil)
		if rec.Code != http.StatusPreconditionFailed {
			t.Fatalf("expected 412, got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid settings are rejected", func(t *testing.T) {
		bad := map[string]any{"enabled": true, "bucket": "", "interval": "24h"}
		rec := doRequest(http.MethodPut, "/api/admin/settings/backup-s3", adminToken, bad)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("save applies to the scheduler and hides the secret", func(t *testing.T) {
		rec := doRequest(http.MethodPut, "/api/admin/settings/backup-s3", adminToken, settings)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("PUT: %d %s", rec.Code, rec.Body.String())
		}

		rec = doRequest(http.MethodGet, "/api/admin/settings/backup-s3", adminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET: %d %s", rec.Code, rec.Body.String())
		}
		if bytes.Contains(rec.Body.Bytes(), []byte("secret-1")) {
			t.Fatal("secret access key must not be returned")
		}
		var pub backup.PublicS3Settings
		if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
			t.Fatal(err)
		}
		if !pub.Enabled || !pub.SecretConfigured || pub.Bucket != "lab-backups" || pub.Interval != "12h0m0s" {
			t.Fatalf("unexpected settings: %+v", pub)
		}

		rec = doRequest(http.MethodGet, "/api/admin/backups/status", adminToken, nil)
		var status struct {
			S3 backup.S3SchedulerStatus `json:"s3_scheduled"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if !status.S3.Enabled || status.S3.Bucket != "lab-backups" || status.S3.NextRunUTC == "" {
			t.Fatalf("scheduler did not pick up the settings: %+v", status.S3)
		}
	})
}
