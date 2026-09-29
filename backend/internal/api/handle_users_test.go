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
		payload := map[string]any{"new_password": "weak"} // Fails complexity check
		rec := doRequest(http.MethodPatch, "/api/auth/me/update-password", accessToken, payload)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for weak password, got %d", rec.Code)
		}
	})

	t.Run("Update Password (Authenticated) - Happy Path", func(t *testing.T) {
		payload := map[string]any{"new_password": "NewValidPassword123!"}
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
