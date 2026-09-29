package auth

import (
	"fmt"
	"log"
)

// EnsureAdminUser checks if at least one admin exists.
// If not, it creates a default admin: admin / admin, flagged to change password
// before using the rest of the application.
func (s *Service) EnsureAdminUser() error {

	count, err := countAdminsRecord(s.db)
	if err != nil {
		return fmt.Errorf("failed to check for admin user: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Println("no admin user found, creating default bootstrap admin...")

	userID, err := s.RegisterUser(
		"Initial Admin",
		"admin", // Default Email/Username
		RoleAdmin,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to create bootstrap admin: %w", err)
	}

	if err := s.UpdateUserPassword(userID, "admin", nil); err != nil {
		return fmt.Errorf("failed to set bootstrap password: %w", err)
	}

	if err := setMustChangePasswordRecord(s.db, userID, true); err != nil {
		return fmt.Errorf("failed to flag bootstrap password change: %w", err)
	}

	log.Println("default admin created (admin/admin). Password change is required on first login.")
	return nil
}
