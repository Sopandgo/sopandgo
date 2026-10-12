package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

func TestPASETOTokenLifecycle(t *testing.T) {
	userID := "user-1234-abcd"
	role := "admin"

	// 1. Generate Token (Happy Path)
	tokenStr, err := auth.GenerateAccessToken(userID, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("Expected token string, got empty")
	}

	// PASETO v4 symmetric tokens always start with this exact prefix
	if !strings.HasPrefix(tokenStr, "v4.local.") {
		t.Errorf("Expected token to be a PASETO v4 local token, got %s", tokenStr)
	}

	// 2. Verify Token (Happy Path)
	claims, err := auth.VerifyToken(tokenStr)
	if err != nil {
		t.Fatalf("Failed to verify valid token: %v", err)
	}
	if claims == nil {
		t.Fatal("Expected claims, got nil")
	}
	if claims.UserID != userID {
		t.Errorf("Expected user ID %q, got %q", userID, claims.UserID)
	}
	if claims.UserRole != role {
		t.Errorf("Expected role %q, got %q", role, claims.UserRole)
	}

	// 3. Unhappy Path: Tampered Token
	// We simulate a hacker intercepting the token and changing the last few characters
	tamperedToken := tokenStr[:len(tokenStr)-5] + "h4ck3d"
	_, err = auth.VerifyToken(tamperedToken)
	if err == nil {
		t.Fatal("CRITICAL: PASETO parser accepted a tampered token!")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("Expected token validation failure, got: %v", err)
	}

	// 4. Unhappy Path: Completely invalid string
	_, err = auth.VerifyToken("just-a-random-string")
	if err == nil {
		t.Fatal("Expected random string to fail verification")
	}

	// 5. Unhappy Path: JWT Token (Downgrade Attack Prevention)
	// Proves PASETO naturally rejects JWTs if someone tries to submit one
	jwtSpoof := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlci0xMjM0LWFiY2QiLCJyb2xlIjoiYWRtaW4ifQ.signature"
	_, err = auth.VerifyToken(jwtSpoof)
	if err == nil {
		t.Fatal("Expected PASETO parser to reject a fake JWT token format")
	}
}

func TestTwoFactorChallenge_SingleUse(t *testing.T) {
	token, err := auth.GenerateTwoFactorChallenge("user-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.VerifyTwoFactorChallenge(token)
	if err != nil || claims.UserID != "user-1" || claims.ID == "" {
		t.Fatalf("verify: %+v %v", claims, err)
	}
	if !auth.ConsumeTwoFactorChallenge(claims) {
		t.Fatal("first consume should succeed")
	}
	if auth.ConsumeTwoFactorChallenge(claims) {
		t.Fatal("second consume must fail")
	}
	if _, err := auth.VerifyTwoFactorChallenge(token); !errors.Is(err, auth.ErrChallengeUsed) {
		t.Fatalf("verify a used challenge: got %v", err)
	}

	// Another challenge for the same user is independent.
	other, _ := auth.GenerateTwoFactorChallenge("user-1")
	if _, err := auth.VerifyTwoFactorChallenge(other); err != nil {
		t.Fatalf("fresh challenge: %v", err)
	}

	// A challenge is not an access token, and an access token is not a challenge.
	if _, err := auth.VerifyToken(token); err == nil {
		t.Fatal("challenge accepted as access token")
	}
	access, _ := auth.GenerateAccessToken("user-1", auth.RoleViewer)
	if _, err := auth.VerifyTwoFactorChallenge(access); err == nil {
		t.Fatal("access token accepted as challenge")
	}
}
