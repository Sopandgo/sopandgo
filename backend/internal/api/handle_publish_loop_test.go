package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_PublishLoopSurfaces(t *testing.T) {
	env := testenv.New(t)

	editorID := "editor-loop-api"
	approverID := "approver-loop-api"
	viewerID := "viewer-loop-api"
	auditorID := "auditor-loop-api"

	env.SeedTestUser(t, editorID, "editor@loop-api.local", auth.RoleEditor)
	env.SeedTestUser(t, approverID, "approver@loop-api.local", auth.RoleApprover)
	env.SeedTestUser(t, viewerID, "viewer@loop-api.local", auth.RoleViewer)
	env.SeedTestUser(t, auditorID, "auditor@loop-api.local", auth.RoleAuditor)

	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	approverToken, _ := auth.GenerateAccessToken(approverID, auth.RoleApprover)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)
	auditorToken, _ := auth.GenerateAccessToken(auditorID, auth.RoleAuditor)

	var ipCounter int
	doJSON := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", ipCounter%200+1)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	publishVersion := func(t *testing.T, title, content, summary string) (sopID, versionID string) {
		t.Helper()
		var err error
		sopID, err = env.SOPService.RegisterSOP(title, &editorID)
		if err != nil {
			t.Fatalf("RegisterSOP: %v", err)
		}
		rec := doJSON(http.MethodPost, "/api/sops/"+sopID, editorToken, map[string]string{
			"content":        content,
			"change_summary": summary,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("create draft: %d %s", rec.Code, rec.Body.String())
		}
		var created struct {
			ID string `json:"id"`
		}
		json.NewDecoder(rec.Body).Decode(&created)
		versionID = created.ID

		rec = doJSON(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/promote", sopID, versionID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("promote: %d %s", rec.Code, rec.Body.String())
		}
		rec = doJSON(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/approve", sopID, versionID), approverToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("approve: %d %s", rec.Code, rec.Body.String())
		}
		return sopID, versionID
	}

	t.Run("diff between consecutive published versions", func(t *testing.T) {
		sopID, v1 := publishVersion(t, "Diff SOP", "line one\n", "Initial publish")

		rec := doJSON(http.MethodPost, "/api/sops/"+sopID, editorToken, map[string]string{
			"content":        "line one\nline two\n",
			"change_summary": "Add second line",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("second draft: %d %s", rec.Code, rec.Body.String())
		}
		var created struct {
			ID string `json:"id"`
		}
		json.NewDecoder(rec.Body).Decode(&created)
		v2 := created.ID
		doJSON(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/promote", sopID, v2), editorToken, nil)
		doJSON(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/approve", sopID, v2), approverToken, nil)

		rec = doJSON(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/diff", sopID, v2), viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("diff: %d %s", rec.Code, rec.Body.String())
		}
		var diff sop.VersionDiff
		json.NewDecoder(rec.Body).Decode(&diff)
		if diff.FromVersionID != v1 {
			t.Errorf("expected from %s, got %s", v1, diff.FromVersionID)
		}
		if len(diff.Lines) == 0 {
			t.Error("expected non-empty diff lines")
		}
	})

	t.Run("recent publishes feed", func(t *testing.T) {
		publishVersion(t, "Activity SOP", "# Activity\n", "Ship activity feed item")
		rec := doJSON(http.MethodGet, "/api/activity/publishes?limit=5", viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("activity: %d %s", rec.Code, rec.Body.String())
		}
		var items []sop.PublishedActivity
		json.NewDecoder(rec.Body).Decode(&items)
		if len(items) == 0 {
			t.Fatal("expected at least one recent publish")
		}
		if items[0].Title == "" || items[0].ChangeSummary == "" {
			t.Errorf("incomplete activity item: %+v", items[0])
		}
	})

	t.Run("training coverage scopes", func(t *testing.T) {
		sopID, _ := publishVersion(t, "Coverage SOP", "# Cover me\n", "Coverage publish")

		rec := doJSON(http.MethodGet, "/api/training/coverage", approverToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("global coverage: %d %s", rec.Code, rec.Body.String())
		}
		rec = doJSON(http.MethodGet, "/api/sops/"+sopID+"/training", approverToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("sop coverage: %d %s", rec.Code, rec.Body.String())
		}
		rec = doJSON(http.MethodGet, "/api/training/coverage", viewerToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("viewer should lack training scope, got %d", rec.Code)
		}
		rec = doJSON(http.MethodGet, "/api/training/coverage", auditorToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("auditor should lack training scope, got %d", rec.Code)
		}
	})

	t.Run("publish notice respects mail mode and failures", func(t *testing.T) {
		env.MockMailSender.SentCount = 0
		env.MockMailSender.SetFail(false)
		if err := env.SMTPSettings.SetMailMode(mail.MailModeSMTP); err != nil {
			t.Fatalf("SetMailMode smtp: %v", err)
		}

		_, _ = publishVersion(t, "Notify SMTP SOP", "# Notify\n", "Notify readers")
		if env.MockMailSender.SentCount < 1 {
			t.Fatalf("expected publish emails in smtp mode, got %d", env.MockMailSender.SentCount)
		}
		if !strings.Contains(env.MockMailSender.LastSubject, "Published:") {
			t.Errorf("unexpected subject %q", env.MockMailSender.LastSubject)
		}

		env.MockMailSender.SentCount = 0
		if err := env.SMTPSettings.SetMailMode(mail.MailModeManualLinks); err != nil {
			t.Fatalf("SetMailMode manual: %v", err)
		}
		_, versionID := publishVersion(t, "Notify Manual SOP", "# Manual\n", "No email in manual mode")
		if env.MockMailSender.SentCount != 0 {
			t.Fatalf("manual_links should skip publish email, got %d sends", env.MockMailSender.SentCount)
		}

		// Failed delivery must not undo publish
		if err := env.SMTPSettings.SetMailMode(mail.MailModeSMTP); err != nil {
			t.Fatalf("SetMailMode smtp again: %v", err)
		}
		env.MockMailSender.SetFail(true)
		sopID, failVersionID := publishVersion(t, "Notify Fail SOP", "# Fail\n", "Delivery fails")
		env.MockMailSender.SetFail(false)

		ver, err := env.SOPService.GetSOPVersionByID(failVersionID)
		if err != nil {
			t.Fatalf("GetSOPVersionByID: %v", err)
		}
		if ver.Status != sop.StatePublished {
			t.Fatalf("expected published despite mail failure, got %s", ver.Status)
		}
		_ = versionID
		_ = sopID
	})
}

func TestAPI_PasswordChangeRequired(t *testing.T) {
	env := testenv.New(t)
	if err := env.AuthService.EnsureAdminUser(); err != nil {
		t.Fatalf("EnsureAdminUser: %v", err)
	}
	admin, _, err := env.AuthService.GetUserByEmail("admin")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if !admin.MustChangePassword {
		t.Fatal("expected must_change_password on bootstrap admin")
	}

	token, _ := auth.GenerateAccessToken(admin.ID, auth.RoleAdmin)
	var ipCounter int
	doJSON := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:1234", ipCounter%200+1)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	rec := doJSON(http.MethodGet, "/api/auth/me", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get me should work: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(http.MethodGet, "/api/sops", nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("sops should be blocked until password change, got %d", rec.Code)
	}

	rec = doJSON(http.MethodPatch, "/api/auth/me/update-password", map[string]string{
		"new_password": "A-strong-boot1!",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("update password: %d %s", rec.Code, rec.Body.String())
	}

	cleared, err := env.AuthService.GetUserByID(admin.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if cleared.MustChangePassword {
		t.Fatal("expected must_change_password cleared")
	}

	// Sessions were revoked; mint a fresh token for the same user
	token, _ = auth.GenerateAccessToken(admin.ID, auth.RoleAdmin)
	rec = doJSON(http.MethodGet, "/api/sops", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("sops should work after password change: %d %s", rec.Code, rec.Body.String())
	}
}
