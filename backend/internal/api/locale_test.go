package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_LocalePreference(t *testing.T) {
	env := testenv.New(t)
	adminID := "admin-locale"
	env.SeedTestUser(t, adminID, "admin@locale.local", auth.RoleAdmin)

	userID, err := env.AuthService.RegisterUser("Locale User", "user@locale.local", auth.RoleEditor, &adminID)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := env.AuthService.UpdateUserPassword(userID, "SuperSecret123!", &adminID); err != nil {
		t.Fatalf("password: %v", err)
	}

	loginBody, _ := json.Marshal(map[string]string{
		"email":    "user@locale.local",
		"password": "SuperSecret123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	env.API.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login %d %s", loginRec.Code, loginRec.Body.String())
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}

	authed := func(method, path string, body any) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	me := authed(http.MethodGet, "/api/auth/me", nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me %d %s", me.Code, me.Body.String())
	}
	var profile struct {
		Locale string `json:"locale"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Locale != "en" {
		t.Fatalf("default locale %q", profile.Locale)
	}

	bad := authed(http.MethodPatch, "/api/auth/me/locale", map[string]string{"locale": "fr"})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for fr, got %d %s", bad.Code, bad.Body.String())
	}

	ok := authed(http.MethodPatch, "/api/auth/me/locale", map[string]string{"locale": "de"})
	if ok.Code != http.StatusOK {
		t.Fatalf("set de %d %s", ok.Code, ok.Body.String())
	}

	me = authed(http.MethodGet, "/api/auth/me", nil)
	if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Locale != "de" {
		t.Fatalf("locale after patch %q", profile.Locale)
	}

	adminLogin, _ := json.Marshal(map[string]string{
		"email":    "admin@locale.local",
		"password": "unused",
	})
	_ = adminLogin
	// Seeded users have unknowable passwords. Use the service, then an admin token via a known password.
	if err := env.AuthService.UpdateUserPassword(adminID, "AdminSecret123!", &adminID); err != nil {
		t.Fatalf("admin password: %v", err)
	}
	adminBody, _ := json.Marshal(map[string]string{
		"email":    "admin@locale.local",
		"password": "AdminSecret123!",
	})
	adminReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(adminBody))
	adminReq.Header.Set("Content-Type", "application/json")
	adminRec := httptest.NewRecorder()
	env.API.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin login %d %s", adminRec.Code, adminRec.Body.String())
	}
	var adminTokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(adminRec.Body.Bytes(), &adminTokens); err != nil {
		t.Fatal(err)
	}
	patch, _ := json.Marshal(map[string]string{"default_locale": "de"})
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings/default-locale", bytes.NewReader(patch))
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.API.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("default locale %d %s", rec.Code, rec.Body.String())
	}
	got, err := env.SMTPSettings.GetDefaultLocale()
	if err != nil {
		t.Fatal(err)
	}
	if got != "de" {
		t.Fatalf("stored default locale %q", got)
	}

	badDefault, _ := json.Marshal(map[string]string{"default_locale": "fr"})
	req = httptest.NewRequest(http.MethodPatch, "/api/admin/settings/default-locale", bytes.NewReader(badDefault))
	req.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	env.API.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for org fr, got %d", rec.Code)
	}
}
