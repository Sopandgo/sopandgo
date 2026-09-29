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

func TestAPI_AuthFlows(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup a Real User with a Real Password Hash
	// We can't use `SeedTestUser` here because we need a real, bcrypt-verifiable password.
	// We'll use the service layer to register and update the password.
	adminID := "admin-actor" // Needed for the audit log
	env.SeedTestUser(t, adminID, "admin@api.local", auth.RoleAdmin)

	targetEmail := "user@auth.local"
	plainPassword := "SuperSecret123!"

	userID, err := env.AuthService.RegisterUser("Test User", targetEmail, auth.RoleEditor, &adminID)
	if err != nil {
		t.Fatalf("Failed to register test user: %v", err)
	}

	err = env.AuthService.UpdateUserPassword(userID, plainPassword, &adminID)
	if err != nil {
		t.Fatalf("Failed to set test user password: %v", err)
	}

	// Helper to fire HTTP requests
	doRequest := func(method, path string, body map[string]any) *httptest.ResponseRecorder {
		var reqBody *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		} else {
			reqBody = bytes.NewReader([]byte{})
		}

		req, _ := http.NewRequest(method, path, reqBody)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	var activeRefreshToken string
	var activeAccessToken string

	// --- RUN TEST CASES ---

	t.Run("Login Failure: Bad Password", func(t *testing.T) {
		payload := map[string]any{
			"email":    targetEmail,
			"password": "WrongPassword!",
		}
		rec := doRequest(http.MethodPost, "/api/auth/login", payload)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for bad password, got %d", rec.Code)
		}
	})

	t.Run("Login Failure: Inactive Account", func(t *testing.T) {
		// Disable the user
		env.AuthService.SetUserActiveStatus(userID, false, &adminID)

		payload := map[string]any{
			"email":    targetEmail,
			"password": plainPassword,
		}
		rec := doRequest(http.MethodPost, "/api/auth/login", payload)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for inactive account, got %d", rec.Code)
		}

		// Re-enable for the rest of the tests
		env.AuthService.SetUserActiveStatus(userID, true, &adminID)
	})

	t.Run("Login Success (Happy Path)", func(t *testing.T) {
		payload := map[string]any{
			"email":    targetEmail,
			"password": plainPassword,
		}
		rec := doRequest(http.MethodPost, "/api/auth/login", payload)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for successful login, got %d", rec.Code)
		}

		var tokens map[string]string
		json.NewDecoder(rec.Body).Decode(&tokens)

		if tokens["access_token"] == "" {
			t.Error("Expected an access_token, got empty")
		}
		if tokens["refresh_token"] == "" {
			t.Error("Expected a refresh_token, got empty")
		}

		// Save the tokens for the next tests
		activeAccessToken = tokens["access_token"]
		activeRefreshToken = tokens["refresh_token"]
	})

	t.Run("Get Me (Using Access Token)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+activeAccessToken)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var user map[string]any
		json.NewDecoder(rec.Body).Decode(&user)
		if user["email"] != targetEmail {
			t.Errorf("Expected email %s, got %v", targetEmail, user["email"])
		}
	})

	t.Run("Get Signature Status (Using Access Token)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/auth/me/signature-status", nil)
		req.Header.Set("Authorization", "Bearer "+activeAccessToken)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for signature status, got %d", rec.Code)
		}

		var statuses []any
		if err := json.NewDecoder(rec.Body).Decode(&statuses); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	})

	t.Run("Refresh Session", func(t *testing.T) {
		payload := map[string]any{
			"refresh_token": activeRefreshToken,
		}
		rec := doRequest(http.MethodPost, "/api/auth/refresh", payload)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for valid refresh, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var response map[string]string
		json.NewDecoder(rec.Body).Decode(&response)

		if response["access_token"] == "" {
			t.Error("Expected a new access_token, got empty")
		}
		// The new access token should be cryptographically different from the old one
		if response["access_token"] == activeAccessToken {
			t.Error("Expected a brand new access token string, but got the exact same one back")
		}
	})

	t.Run("Logout (Revoke Session)", func(t *testing.T) {
		payload := map[string]any{
			"refresh_token": activeRefreshToken,
		}
		rec := doRequest(http.MethodPost, "/api/auth/logout", payload)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content for logout, got %d", rec.Code)
		}
	})

	t.Run("Refresh Session Failure (After Logout)", func(t *testing.T) {
		payload := map[string]any{
			"refresh_token": activeRefreshToken, // Trying to use the token we just destroyed
		}
		rec := doRequest(http.MethodPost, "/api/auth/refresh", payload)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for revoked token, got %d", rec.Code)
		}
	})
}
