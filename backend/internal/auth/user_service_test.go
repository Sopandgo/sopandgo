package auth_test

import (
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_RegisterUser(t *testing.T) {
	env := testenv.New(t)

	// We'll use a dummy actor for the audit log
	actorID := "system-admin"
	env.SeedTestUser(t, actorID, "admin@demo.local", "admin")

	// 1. Happy Path & Data Normalization
	// We intentionally mess up the email to ensure strings.ToLower and TrimSpace work
	dirtyEmail := "  NeW-uSeR@DeMo.Local  "
	displayName := "  New User  "

	userID, err := env.AuthService.RegisterUser(displayName, dirtyEmail, "viewer", &actorID)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	if userID == "" {
		t.Fatal("Expected valid UUID, got empty string")
	}

	// 2. Verify Database State
	user, err := env.AuthService.GetUserByID(userID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	// Email should be perfectly clean and lowercase
	expectedEmail := "new-user@demo.local"
	if user.Email != expectedEmail {
		t.Errorf("Expected email %q, got %q", expectedEmail, user.Email)
	}

	// Display Name should just be trimmed, preserving case
	expectedName := "New User"
	if user.DisplayName != expectedName {
		t.Errorf("Expected display name %q, got %q", expectedName, user.DisplayName)
	}
}

func TestService_UserGetters(t *testing.T) {
	env := testenv.New(t)

	userID := "target-user-1"
	email := "target@demo.local"
	env.SeedTestUser(t, userID, email, "editor")

	// 1. GetUserByEmail
	user, hash, err := env.AuthService.GetUserByEmail(email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("Expected user ID %q, got %q", userID, user.ID)
	}
	if hash != "dummy-hash" { // Matches what SeedTestUser inserts
		t.Errorf("Expected 'dummy-hash', got %q", hash)
	}

	// 2. GetUserRole
	role, err := env.AuthService.GetUserRole(userID)
	if err != nil {
		t.Fatalf("GetUserRole failed: %v", err)
	}
	if role != "editor" {
		t.Errorf("Expected role 'editor', got %q", role)
	}

	// 3. GetUserActiveStatus
	isActive, err := env.AuthService.GetUserActiveStatus(userID)
	if err != nil {
		t.Fatalf("GetUserActiveStatus failed: %v", err)
	}
	if !isActive {
		t.Error("Expected seeded user to be active, but was false")
	}

	// 4. ListUsers
	users, err := env.AuthService.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestService_SecurityPolicies(t *testing.T) {
	env := testenv.New(t)

	adminID := "admin-1"
	targetID := "user-1"

	env.SeedTestUser(t, adminID, "admin@demo.local", "admin")
	env.SeedTestUser(t, targetID, "user@demo.local", "viewer")

	// 1. UpdateUserRole - Policy Check
	err := env.AuthService.UpdateUserRole(adminID, "viewer", &adminID)
	if err == nil || !strings.Contains(err.Error(), "users cannot modify their own roles") {
		t.Fatalf("Expected self-demotion to fail, got: %v", err)
	}

	// 2. UpdateUserRole - Happy Path
	err = env.AuthService.UpdateUserRole(targetID, "editor", &adminID)
	if err != nil {
		t.Fatalf("Admin failed to update target user's role: %v", err)
	}
	newRole, _ := env.AuthService.GetUserRole(targetID)
	if newRole != "editor" {
		t.Errorf("Expected target user to be an editor, got %q", newRole)
	}

	// 3. SetUserActiveStatus - Policy Check
	err = env.AuthService.SetUserActiveStatus(adminID, false, &adminID)
	if err == nil {
		t.Fatal("Expected self-disable to fail, but it succeeded")
	}

	// 4. SetUserActiveStatus - Happy Path
	err = env.AuthService.SetUserActiveStatus(targetID, false, &adminID)
	if err != nil {
		t.Fatalf("Admin failed to disable target user: %v", err)
	}
	isActive, _ := env.AuthService.GetUserActiveStatus(targetID)
	if isActive {
		t.Error("Expected target user to be disabled, but was active")
	}
}

func TestService_PasswordResetFlow(t *testing.T) {
	env := testenv.New(t)

	userID := "reset-user-1"
	env.SeedTestUser(t, userID, "reset@demo.local", "viewer")

	// 1. Generate Token
	rawToken, user, err := env.AuthService.GeneratePasswordResetToken(userID)
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken failed: %v", err)
	}
	if rawToken == "" {
		t.Fatal("Expected non-empty raw token")
	}
	if user.Email != "reset@demo.local" {
		t.Errorf("Expected user email 'reset@demo.local', got %q", user.Email)
	}

	// 2. Reset Password (Happy Path)
	newPassword := "SuperSecret123!"
	err = env.AuthService.ResetPassword(rawToken, newPassword)
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// 3. Burned Token Guarantee (Single Use)
	// Trying to use the exact same token again MUST fail.
	err = env.AuthService.ResetPassword(rawToken, "AnotherPassword")
	if err == nil {
		t.Fatal("CRITICAL: Token was not burned! Allowed password reset twice with same token.")
	}
	if !strings.Contains(err.Error(), "invalid or expired link") {
		t.Errorf("Expected 'invalid or expired link' error, got: %v", err)
	}

	// 4. Verify Password Was Actually Changed
	// We check this by fetching the user and comparing the hash
	// Note: We don't have a direct "CheckPassword" function exported here,
	// but we can prove the DB hash is no longer "dummy-hash".
	_, newHash, err := env.AuthService.GetUserByEmail("reset@demo.local")
	if err != nil {
		t.Fatalf("Failed to fetch user after reset: %v", err)
	}
	if newHash == "dummy-hash" {
		t.Error("Database password hash was not updated during reset")
	}
}
