package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/google/uuid"
)

// RegisterUser handles hashing, creation, and logging in a SINGLE transaction.
func (s *Service) RegisterUser(
	displayName, email, role string,
	actorUserID *string,
) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Prepare Data (Non-Transactional)
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}
	hash := hex.EncodeToString(entropy)

	id := uuid.NewString()
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 2. Start Transaction
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// 3. Insert Record
	if err := createUserRecord(tx, id, displayName, email, hash, role, createdAt); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// 4. Audit Log
	err = s.auditLogger.Log(
		tx,
		audit.EventUserCreated,
		audit.EntityUser,
		id,
		actorUserID,
		map[string]string{
			"display_name": displayName,
			"email":        email,
			"role":         role,
			"auth_method":  "invite_link",
		},
	)
	if err != nil {
		return "", err
	}

	// 5. Commit
	return id, tx.Commit()
}

// GetUserByID retrieves a user.
func (s *Service) GetUserByID(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getUserByIDRecord(s.db, id)
}

// GetUserByEmail retrieves a user and their password hash (for login only).
func (s *Service) GetUserByEmail(email string) (*User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	s.mu.RLock()
	defer s.mu.RUnlock()

	return getUserByEmailRecord(s.db, email)
}

// GetUserRole retrieves just the role string.
func (s *Service) GetUserRole(userID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getUserRoleRecord(s.db, userID)
}

// GetUserActiveStatus retrieves just the active boolean.
func (s *Service) GetUserActiveStatus(userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return getUserActiveStatusRecord(s.db, userID)
}

// ListUsers is the service-layer entry point for retrieving users.
// Currently just a pass-through
func (s *Service) ListUsers() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return listUsersRecord(s.db)
}

// UpdateUserRole handles logic checks, update, session revocation, and logging.
func (s *Service) UpdateUserRole(
	userID string,
	newRole string,
	actorUserID *string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Policy Check
	if actorUserID != nil && *actorUserID == userID {
		return fmt.Errorf("security policy: users cannot modify their own roles")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get Old Role (Read)
	oldRole, err := getUserRoleRecord(tx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch current role: %w", err)
	}

	if oldRole == newRole {
		return nil // No change needed
	}

	// 2. Update Role (Write)
	if err := updateUserRoleRecord(tx, userID, newRole); err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	// 3. Revoke Sessions (Write)
	if err := revokeUserRefreshTokensRecord(tx, userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	// 4. Audit
	err = s.auditLogger.Log(
		tx,
		audit.EventUserRoleUpdated,
		audit.EntityUser,
		userID,
		actorUserID,
		map[string]any{
			"from":             oldRole,
			"to":               newRole,
			"sessions_revoked": true, // This will now be stored as a proper JSON boolean
		},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SetUserActiveStatus handles status change and session killing.
func (s *Service) SetUserActiveStatus(
	userID string,
	active bool,
	actorUserID *string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if actorUserID != nil && *actorUserID == userID {
		return ErrSelfDisable
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get Old Status
	isActive, err := getUserActiveStatusRecord(tx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch status: %w", err)
	}

	if isActive == active {
		return nil
	}

	// 2. Update Status
	statusInt := 0
	if active {
		statusInt = 1
	}
	if err := updateUserStatusRecord(tx, userID, statusInt); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// 3. Revoke Sessions if disabling
	if !active {
		if err := revokeUserRefreshTokensRecord(tx, userID); err != nil {
			return fmt.Errorf("failed to revoke sessions: %w", err)
		}
	}

	// 4. Audit
	err = s.auditLogger.Log(
		tx,
		audit.EventUserStatusChanged,
		audit.EntityUser,
		userID,
		actorUserID,
		map[string]any{
			"from": isActive,
			"to":   active,
		},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Service) UpdateUserPassword(
	userID string,
	newPlainPassword string,
	actorUserID *string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, err := HashPassword(newPlainPassword)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Update Password
	if err := updateUserPasswordRecord(tx, userID, hash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 2. Revoke Sessions (Security Best Practice)
	if err := revokeUserRefreshTokensRecord(tx, userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	// 3. Audit
	err = s.auditLogger.Log(
		tx,
		audit.EventUserPasswordUpdated,
		audit.EntityUser,
		userID,
		actorUserID,
		map[string]any{
			"reason":           "password_reset",
			"sessions_revoked": true,
		},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GeneratePasswordResetToken creates a secure token for a specific user.
// Returns:
// - rawToken: The secret string to put in the email link.
// - user: The user object (so you have the Email and DisplayName for the mailer).
func (s *Service) GeneratePasswordResetToken(userID string) (string, *User, error) {
	// 1. Find the User
	// We need the user struct because the Controller will need s.Email to send the mail.
	user, err := getUserByIDRecord(s.db, userID)
	if err != nil {
		return "", nil, fmt.Errorf("user not found: %w", err)
	}

	// 2. Generate Random Token (32 bytes of entropy)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("failed to generate entropy: %w", err)
	}
	rawToken := hex.EncodeToString(b)

	// 3. Hash it for the Database
	// We strictly store ONLY the hash. If the DB is leaked, the tokens are useless.
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// 4. Save to DB
	// Since an admin triggered this (it might be an invite), we typically give
	// a longer expiration window (e.g., 24 hours) than a self-service reset.
	expiresAt := time.Now().Add(24 * time.Hour)

	if err := saveResetTokenRecord(s.db, user.ID, tokenHash, expiresAt); err != nil {
		return "", nil, fmt.Errorf("failed to save token: %w", err)
	}

	// Return rawToken (for the link) and user (for the email address)
	return rawToken, user, nil
}

// ResetPassword validates the token and calls UpdateUserPassword
func (s *Service) ResetPassword(rawToken, newPlainPassword string) error {
	// 1. Hash the incoming token (security best practice)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	// 2. Lookup the token
	userID, expiresAt, err := getResetTokenRecord(s.db, tokenHash)
	if err != nil {
		return errors.New("invalid or expired link")
	}

	// 3. Check Expiration
	if time.Now().After(expiresAt) {
		// Cleanup expired token immediately
		deleteResetTokenRecord(s.db, userID)
		return errors.New("link has expired")
	}

	// 4. Update user password
	if err := s.UpdateUserPassword(userID, newPlainPassword, nil); err != nil {
		return err
	}

	// 5. Burn the Token (Single Use Guarantee)
	deleteResetTokenRecord(s.db, userID)

	return nil
}
