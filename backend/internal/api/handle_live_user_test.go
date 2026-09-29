package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_LiveUserAuthorization(t *testing.T) {
	env := testenv.New(t)

	editorID := "live-editor-1"
	actorID := "live-admin-1"
	env.SeedTestUser(t, editorID, "editor@live.local", auth.RoleEditor)
	env.SeedTestUser(t, actorID, "admin@live.local", auth.RoleAdmin)

	// Token claims admin, but the stored role is editor.
	staleAdminToken, err := auth.GenerateAccessToken(editorID, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	editorToken, err := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	if err != nil {
		t.Fatal(err)
	}

	var ipCounter int
	doGet := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	t.Run("Stale admin claim uses the stored role", func(t *testing.T) {
		rec := doGet("/api/admin/users", staleAdminToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for demoted role, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Inactive user is rejected on the next request", func(t *testing.T) {
		if err := env.AuthService.SetUserActiveStatus(editorID, false, &actorID); err != nil {
			t.Fatal(err)
		}
		rec := doGet("/api/sops", editorToken)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for inactive user, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
