package auth_test

import (
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestService_EnsureAdminUser(t *testing.T) {
	env := testenv.New(t)

	// 1. Initial State: Prove the database has no users.
	users, err := env.AuthService.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("Expected empty database, found %d users", len(users))
	}

	// 2. Action: Run EnsureAdminUser on an empty DB
	err = env.AuthService.EnsureAdminUser()
	if err != nil {
		t.Fatalf("EnsureAdminUser failed on empty DB: %v", err)
	}

	// 3. Verify: Bootstrap admin was created successfully
	adminUser, hash, err := env.AuthService.GetUserByEmail("admin")
	if err != nil {
		t.Fatalf("Failed to fetch bootstrap admin: %v", err)
	}

	// Verify the role is explicitly 'admin'
	role, _ := env.AuthService.GetUserRole(adminUser.ID)
	if role != auth.RoleAdmin {
		t.Errorf("Expected role %q, got %q", auth.RoleAdmin, role)
	}

	// Verify the default password was correctly hashed and set to "admin"
	err = auth.VerifyPassword("admin", hash)
	if err != nil {
		t.Errorf("Bootstrap admin password is not 'admin': %v", err)
	}

	if !adminUser.MustChangePassword {
		t.Errorf("expected bootstrap admin to require a password change")
	}

	// Changing the password clears the flag
	if err := env.AuthService.UpdateUserPassword(adminUser.ID, "A-strong-boot1!", &adminUser.ID); err != nil {
		t.Fatalf("UpdateUserPassword failed: %v", err)
	}
	cleared, err := env.AuthService.GetUserByID(adminUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID after password change: %v", err)
	}
	if cleared.MustChangePassword {
		t.Errorf("expected must_change_password to clear after password update")
	}

	// 4. Action: Run EnsureAdminUser AGAIN (Idempotency check)
	// If it tries to create the user again, it will fail the UNIQUE constraint on the email.
	err = env.AuthService.EnsureAdminUser()
	if err != nil {
		t.Fatalf("EnsureAdminUser failed on second run (not idempotent): %v", err)
	}

	// Verify it returned early and didn't insert a second user
	users, _ = env.AuthService.ListUsers()
	if len(users) != 1 {
		t.Errorf("Expected exactly 1 user in DB after second run, got %d", len(users))
	}
}
