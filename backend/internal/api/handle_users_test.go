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

func TestAPI_PasswordUpdates(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup User
	userID := "user-pw-1"
	userEmail := "password_test@api.local"
	env.SeedTestUser(t, userID, userEmail, auth.RoleEditor)

	// Set an initial password so we can test login later
	env.AuthService.UpdateUserPassword(userID, "InitialPass123!", &userID)

	// Generate a standard PASETO access token for the authenticated routes
	accessToken, _ := auth.GenerateAccessToken(userID, auth.RoleEditor)

	var ipCounter int

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

		// FIX: Spoof a unique IP address for every single request
		// to prevent the Rate Limiter from blocking our test suite!
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

	var validResetToken string

	// --- RUN TEST CASES ---

	t.Run("Update Password (Authenticated) - Validation Failure", func(t *testing.T) {
		payload := map[string]any{
			"current_password": "InitialPass123!",
			"new_password":     "weak", // Fails complexity check
		}
		rec := doRequest(http.MethodPatch, "/api/auth/me/update-password", accessToken, payload)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for weak password, got %d", rec.Code)
		}
	})

	t.Run("Update Password (Authenticated) - Missing Current Password", func(t *testing.T) {
		payload := map[string]any{"new_password": "NewValidPassword123!"}
		rec := doRequest(http.MethodPatch, "/api/auth/me/update-password", accessToken, payload)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden without current password, got %d", rec.Code)
		}
	})

	t.Run("Update Password (Authenticated) - Wrong Current Password", func(t *testing.T) {
		payload := map[string]any{
			"current_password": "NotMyPassword123!",
			"new_password":     "NewValidPassword123!",
		}
		rec := doRequest(http.MethodPatch, "/api/auth/me/update-password", accessToken, payload)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for wrong current password, got %d", rec.Code)
		}

		// The old password must still work
		loginPayload := map[string]any{"email": userEmail, "password": "InitialPass123!"}
		loginRec := doRequest(http.MethodPost, "/api/auth/login", "", loginPayload)
		if loginRec.Code != http.StatusOK {
			t.Error("Password changed despite a wrong current password!")
		}
	})

	t.Run("Update Password (Authenticated) - Happy Path", func(t *testing.T) {
		payload := map[string]any{
			"current_password": "InitialPass123!",
			"new_password":     "NewValidPassword123!",
		}
		rec := doRequest(http.MethodPatch, "/api/auth/me/update-password", accessToken, payload)

		if rec.Code != http.StatusNoContent {
			t.Errorf("Expected 204 No Content, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify the password actually changed by trying to log in
		loginPayload := map[string]any{"email": userEmail, "password": "NewValidPassword123!"}
		loginRec := doRequest(http.MethodPost, "/api/auth/login", "", loginPayload)
		if loginRec.Code != http.StatusOK {
			t.Error("Failed to log in with the newly updated password!")
		}
	})

	t.Run("Reset Password - Generate Token", func(t *testing.T) {
		// We use the service layer to generate a token, simulating the Admin/Forgot Password flow
		token, _, err := env.AuthService.GeneratePasswordResetToken(userID)
		if err != nil {
			t.Fatalf("Failed to generate reset token: %v", err)
		}
		validResetToken = token
	})

	t.Run("Reset Password - Validation Failure", func(t *testing.T) {
		payload := map[string]any{
			"token":        validResetToken,
			"new_password": "short", // Fails complexity
		}
		rec := doRequest(http.MethodPatch, "/api/auth/reset-password", "", payload)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for weak password, got %d", rec.Code)
		}
	})

	// Reset links prove identity with the token, so they never ask for the
	// current password.
	t.Run("Reset Password - Happy Path", func(t *testing.T) {
		payload := map[string]any{
			"token":        validResetToken,
			"new_password": "RecoveredPassword123!",
		}
		rec := doRequest(http.MethodPatch, "/api/auth/reset-password", "", payload)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// Verify login works with the recovered password
		loginPayload := map[string]any{"email": userEmail, "password": "RecoveredPassword123!"}
		loginRec := doRequest(http.MethodPost, "/api/auth/login", "", loginPayload)
		if loginRec.Code != http.StatusOK {
			t.Error("Failed to log in with the recovered password!")
		}
	})

	t.Run("Reset Password - Token Burned (Single Use Check)", func(t *testing.T) {
		// Try to use the exact same token again
		payload := map[string]any{
			"token":        validResetToken,
			"new_password": "HackerPassword123!",
		}
		rec := doRequest(http.MethodPatch, "/api/auth/reset-password", "", payload)

		// The handler should return 401 Unauthorized because the token no longer exists in the DB
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("CRITICAL: Token was not burned! Expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Reset Password - Invalid/Fake Token", func(t *testing.T) {
		payload := map[string]any{
			"token":        "fake-token-string",
			"new_password": "ValidPassword123!",
		}
		rec := doRequest(http.MethodPatch, "/api/auth/reset-password", "", payload)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for fake token, got %d", rec.Code)
		}
	})
}

func TestAPI_SignOutOtherSessions(t *testing.T) {
	env := testenv.New(t)

	userID := "user-sessions-1"
	env.SeedTestUser(t, userID, "sessions_test@api.local", auth.RoleEditor)
	otherID := "user-sessions-2"
	env.SeedTestUser(t, otherID, "sessions_other@api.local", auth.RoleEditor)

	current, _ := env.AuthService.CreateSession(userID)
	labPC, _ := env.AuthService.CreateSession(userID)
	otherUser, _ := env.AuthService.CreateSession(otherID)

	accessToken, _ := auth.GenerateAccessToken(userID, auth.RoleEditor)

	var ipCounter int
	signOutOthers := func(refreshToken string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/me/sessions/sign-out-others", bytes.NewReader(b))
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", ipCounter)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	t.Run("Rejects a session that belongs to someone else", func(t *testing.T) {
		rec := signOutOthers(otherUser)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d", rec.Code)
		}
		if _, _, err := env.AuthService.ValidateSession(labPC); err != nil {
			t.Error("Sessions were revoked for a request with a foreign token")
		}
	})

	t.Run("Revokes other sessions and keeps the current one", func(t *testing.T) {
		rec := signOutOthers(current)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var body struct {
			SessionsRevoked int `json:"sessions_revoked"`
		}
		json.NewDecoder(rec.Body).Decode(&body)
		if body.SessionsRevoked != 1 {
			t.Errorf("Expected 1 revoked session, got %d", body.SessionsRevoked)
		}

		if _, _, err := env.AuthService.ValidateSession(current); err != nil {
			t.Errorf("Current session was revoked: %v", err)
		}
		if _, _, err := env.AuthService.ValidateSession(labPC); err == nil {
			t.Error("Other session is still active")
		}
		if _, _, err := env.AuthService.ValidateSession(otherUser); err != nil {
			t.Errorf("Another person's session was revoked: %v", err)
		}
	})
}
