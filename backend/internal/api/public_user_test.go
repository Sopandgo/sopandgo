package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// TestAPI_OtherUsersArePublicOnly: SOP responses show other people by id,
// display name and avatar only. Their email, role, 2FA status and creation
// date stay in user administration and in each user's own /api/auth/me.
func TestAPI_OtherUsersArePublicOnly(t *testing.T) {
	env := testenv.New(t)

	editor, approver, reader := "pub-editor", "pub-approver", "pub-reader"
	env.SeedTestUser(t, editor, "editor@public.local", auth.RoleEditor)
	env.SeedTestUser(t, approver, "approver@public.local", auth.RoleApprover)
	env.SeedTestUser(t, reader, "reader@public.local", auth.RoleViewer)
	admin := "pub-admin"
	env.SeedTestUser(t, admin, "admin@public.local", auth.RoleAdmin)

	readerToken, _ := auth.GenerateAccessToken(reader, auth.RoleViewer)
	approverToken, _ := auth.GenerateAccessToken(approver, auth.RoleApprover)
	adminToken, _ := auth.GenerateAccessToken(admin, auth.RoleAdmin)

	sopID, _ := env.SOPService.RegisterSOP("Public User SOP", &editor)
	v1, _, err := env.SOPService.RegisterSOPVersion(sopID, "# One", "first", editor)
	if err != nil {
		t.Fatalf("RegisterSOPVersion: %v", err)
	}
	if err := env.SOPService.PromoteSOPVersion(v1, editor); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := env.SOPService.ApproveSOPVersion(v1, approver); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if _, err := env.SOPService.AddAcknowledgment(v1, reader, "reader"); err != nil {
		t.Fatalf("AddAcknowledgment: %v", err)
	}
	v2, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Two", "second", editor)
	if err != nil {
		t.Fatalf("RegisterSOPVersion v2: %v", err)
	}
	if err := env.SOPService.PromoteSOPVersion(v2, editor); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}
	if err := env.SOPService.RejectSOPVersion(v2, approver, "not yet"); err != nil {
		t.Fatalf("Reject v2: %v", err)
	}

	var ip int
	get := func(path, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		ip++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ip)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	private := []string{
		"editor@public.local", "approver@public.local", "reader@public.local",
		"two_factor_enabled", "must_change_password", `"role"`, `"email"`,
	}
	paths := []struct {
		path  string
		token string
	}{
		{fmt.Sprintf("/api/sops/%s/versions/%s/acks", sopID, v1), readerToken},
		{fmt.Sprintf("/api/sops/%s/versions/%s/summary", sopID, v1), readerToken},
		{fmt.Sprintf("/api/sops/%s/versions/%s/summary", sopID, v2), approverToken},
		{fmt.Sprintf("/api/sops/%s/version-latest", sopID), readerToken},
		{"/api/activity/publishes", readerToken},
		{"/api/training/coverage", approverToken},
		{fmt.Sprintf("/api/sops/%s/training", sopID), approverToken},
	}
	for _, p := range paths {
		rec := get(p.path, p.token)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", p.path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, s := range private {
			if strings.Contains(body, s) {
				t.Errorf("GET %s exposes %s: %s", p.path, s, body)
			}
		}
	}

	t.Run("acknowledgment users have exactly the public fields", func(t *testing.T) {
		rec := get(fmt.Sprintf("/api/sops/%s/versions/%s/acks", sopID, v1), readerToken)
		var acks []struct {
			User map[string]any `json:"user"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &acks); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(acks) == 0 {
			t.Fatal("expected acknowledgments")
		}
		for _, a := range acks {
			keys := make([]string, 0, len(a.User))
			for k := range a.User {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			got := strings.Join(keys, ",")
			if got != "display_name,has_avatar,id" && got != "avatar_content_hash,display_name,has_avatar,id" {
				t.Errorf("ack user keys = %s, want id, display_name, has_avatar[, avatar_content_hash]", got)
			}
		}
	})

	t.Run("own profile and user administration keep full records", func(t *testing.T) {
		me := get("/api/auth/me", readerToken)
		if !strings.Contains(me.Body.String(), "reader@public.local") {
			t.Errorf("/api/auth/me lost the email: %s", me.Body.String())
		}
		users := get("/api/admin/users", adminToken)
		if !strings.Contains(users.Body.String(), "editor@public.local") {
			t.Errorf("/api/admin/users lost emails: %s", users.Body.String())
		}
	})
}
