package auth_test

import (
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_SessionLifecycle(t *testing.T) {
	env := testenv.New(t)
	userID := "session-user-1"
	role := "editor"

	// 1. Setup
	env.SeedTestUser(t, userID, "session@demo.local", role)

	// 2. Create Session
	tokenID, err := env.AuthService.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if len(tokenID) < 10 {
		t.Fatalf("Expected a secure token UUID, got %q", tokenID)
	}

	// 3. Validate Session (Happy Path)
	valUserID, valRole, err := env.AuthService.ValidateSession(tokenID)
	if err != nil {
		t.Fatalf("ValidateSession failed on valid token: %v", err)
	}
	if valUserID != userID {
		t.Errorf("Expected user ID %q, got %q", userID, valUserID)
	}
	if valRole != role {
		t.Errorf("Expected role %q, got %q", role, valRole)
	}

	// 4. Validate Invalid Session (Unhappy Path)
	_, _, err = env.AuthService.ValidateSession("fake-made-up-token")
	if err == nil {
		t.Fatal("Expected validation to fail for a fake token, but it succeeded")
	}
}

func TestService_RevokeSession(t *testing.T) {
	env := testenv.New(t)
	userID := "logout-user-1"
	env.SeedTestUser(t, userID, "logout@demo.local", "viewer")

	// 1. Create a session
	tokenID, _ := env.AuthService.CreateSession(userID)

	// 2. Revoke it (Simulate Logout)
	err := env.AuthService.RevokeSession(tokenID, &userID)
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	// 3. Attempt to validate the revoked session
	_, _, err = env.AuthService.ValidateSession(tokenID)
	if err == nil {
		t.Fatal("CRITICAL: Revoked session was still successfully validated!")
	}
}

func TestService_RevokeAllUserSessions(t *testing.T) {
	env := testenv.New(t)
	adminID := "admin-1"
	targetUserID := "hacked-user-1"
	safeUserID := "safe-user-1"

	env.SeedTestUser(t, adminID, "admin@demo.local", "admin")
	env.SeedTestUser(t, targetUserID, "hacked@demo.local", "editor")
	env.SeedTestUser(t, safeUserID, "safe@demo.local", "viewer")

	// 1. Create multiple sessions for the target user (e.g., Phone and Laptop)
	targetToken1, _ := env.AuthService.CreateSession(targetUserID)
	targetToken2, _ := env.AuthService.CreateSession(targetUserID)

	// 2. Create a session for a completely different user
	safeToken, _ := env.AuthService.CreateSession(safeUserID)

	// 3. Admin revokes all sessions for the target user
	err := env.AuthService.RevokeAllUserSessions(targetUserID, &adminID)
	if err != nil {
		t.Fatalf("RevokeAllUserSessions failed: %v", err)
	}

	// 4. Assert target user's sessions are dead
	if _, _, err := env.AuthService.ValidateSession(targetToken1); err == nil {
		t.Error("Target's first session is still alive")
	}
	if _, _, err := env.AuthService.ValidateSession(targetToken2); err == nil {
		t.Error("Target's second session is still alive")
	}

	// 5. Assert the safe user was untouched
	if _, _, err := env.AuthService.ValidateSession(safeToken); err != nil {
		t.Errorf("Safe user's session was accidentally revoked: %v", err)
	}
}

func TestService_RevokeAllSessions_GlobalKillSwitch(t *testing.T) {
	env := testenv.New(t)
	adminID := "super-admin"
	userA := "user-a"
	userB := "user-b"

	env.SeedTestUser(t, adminID, "admin@demo.local", "admin")
	env.SeedTestUser(t, userA, "a@demo.local", "editor")
	env.SeedTestUser(t, userB, "b@demo.local", "viewer")

	// 1. Create sessions for everyone
	tokenA, _ := env.AuthService.CreateSession(userA)
	tokenB, _ := env.AuthService.CreateSession(userB)

	// 2. Pull the global kill switch!
	err := env.AuthService.RevokeAllSessions("Suspected database breach", &adminID)
	if err != nil {
		t.Fatalf("RevokeAllSessions failed: %v", err)
	}

	// 3. Assert NO sessions survive
	if _, _, err := env.AuthService.ValidateSession(tokenA); err == nil {
		t.Error("User A's session survived the global kill switch")
	}
	if _, _, err := env.AuthService.ValidateSession(tokenB); err == nil {
		t.Error("User B's session survived the global kill switch")
	}
}
