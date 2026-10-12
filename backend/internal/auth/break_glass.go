package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

var (
	ErrBreakGlassUserNotFound = errors.New("no account with that email")
	ErrBreakGlassNotAdmin     = errors.New("break-glass only resets admin accounts")
)

// Reasons recorded on admin_break_glass_failed.
const (
	BreakGlassFailUserNotFound = "user_not_found"
	BreakGlassFailNotAdmin     = "not_admin"
	BreakGlassFailAborted      = "aborted_by_operator"
)

// BreakGlassContext says where a break-glass run came from. It is written to
// the audit log so a reviewer can tell who used the host terminal.
type BreakGlassContext struct {
	Hostname string
	OSUser   string
}

// BreakGlassResult describes what a reset changed. The temporary password is
// shown to the operator once and is never logged.
type BreakGlassResult struct {
	UserID            string
	Email             string
	TemporaryPassword string
	Reactivated       bool
	SessionsRevoked   int64
	Cleared           []string
}

// temporaryPasswordAlphabet has lower, upper, and digits, so every password
// passes ValidatePassword's three-class rule once each class is present.
const (
	tempLower = "abcdefghijkmnpqrstuvwxyz"
	tempUpper = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	tempDigit = "23456789"
)

func newTemporaryPassword() (string, error) {
	for {
		pw, err := randomString(tempLower+tempUpper+tempDigit, 24)
		if err != nil {
			return "", err
		}
		if ValidatePassword(pw) == nil {
			return pw, nil
		}
	}
}

// ListAdmins returns every admin account, for the recover-admin command.
func (s *Service) ListAdmins() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return listAdminsRecord(s.db)
}

// BreakGlassResetAdmin is the host-only recovery for a locked-out admin. It is
// called by the recover-admin command in the backend binary and has no HTTP
// route. In one transaction it removes every second factor, sets a temporary
// password that must be changed at the next sign-in, reactivates the account,
// signs out every session, and writes admin_break_glass_reset. If the audit
// write fails, nothing changes.
func (s *Service) BreakGlassResetAdmin(email string, bg BreakGlassContext) (*BreakGlassResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	s.mu.Lock()
	defer s.mu.Unlock()

	user, _, err := getUserByEmailRecord(s.db, email)
	if errors.Is(err, sql.ErrNoRows) {
		s.logBreakGlassFailure("", email, BreakGlassFailUserNotFound, bg)
		return nil, ErrBreakGlassUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if user.Role != RoleAdmin {
		s.logBreakGlassFailure(user.ID, email, BreakGlassFailNotAdmin, bg)
		return nil, ErrBreakGlassNotAdmin
	}

	tempPassword, err := newTemporaryPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate temporary password: %w", err)
	}
	hash, err := HashPassword(tempPassword)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	totpRows, recoveryRows, emailRows, err := deleteTwoFactorRecords(tx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to remove second factor: %w", err)
	}
	cleared := []string{}
	if totpRows > 0 {
		cleared = append(cleared, "totp")
	}
	if recoveryRows > 0 {
		cleared = append(cleared, "recovery_codes")
	}
	if emailRows > 0 {
		cleared = append(cleared, "email_codes")
	}

	if err := updateUserPasswordRecord(tx, user.ID, hash); err != nil {
		return nil, fmt.Errorf("failed to set temporary password: %w", err)
	}
	if err := setMustChangePasswordRecord(tx, user.ID, true); err != nil {
		return nil, fmt.Errorf("failed to require password change: %w", err)
	}
	if err := deleteResetTokenRecord(tx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to remove reset link: %w", err)
	}
	reactivated, err := reactivateUserRecord(tx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to reactivate account: %w", err)
	}
	sessions, err := countActiveUserSessionsRecord(tx, user.ID)
	if err != nil {
		return nil, err
	}
	if err := revokeUserRefreshTokensRecord(tx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke sessions: %w", err)
	}

	if err := s.auditLogger.Log(tx, audit.EventAdminBreakGlassReset, audit.EntityUser, user.ID, nil, map[string]any{
		"via":              "cli",
		"command":          "recover-admin",
		"email":            user.Email,
		"hostname":         bg.Hostname,
		"os_user":          bg.OSUser,
		"cleared":          cleared,
		"password_reset":   true,
		"must_change":      true,
		"reactivated":      reactivated,
		"sessions_revoked": sessions,
	}); err != nil {
		return nil, fmt.Errorf("audit write failed, nothing was changed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BreakGlassResult{
		UserID:            user.ID,
		Email:             user.Email,
		TemporaryPassword: tempPassword,
		Reactivated:       reactivated,
		SessionsRevoked:   sessions,
		Cleared:           cleared,
	}, nil
}

// LogBreakGlassAborted records a run the operator cancelled at the prompt.
func (s *Service) LogBreakGlassAborted(email string, bg BreakGlassContext) {
	email = strings.ToLower(strings.TrimSpace(email))
	userID := ""
	if user, _, err := getUserByEmailRecord(s.db, email); err == nil {
		userID = user.ID
	}
	s.logBreakGlassFailure(userID, email, BreakGlassFailAborted, bg)
}

// logBreakGlassFailure writes admin_break_glass_failed. A failed write is
// reported on stderr by the command and in the server log, never ignored.
func (s *Service) logBreakGlassFailure(userID, email, reason string, bg BreakGlassContext) {
	entityType, entityID := audit.EntityUser, userID
	if userID == "" {
		entityType, entityID = audit.EntitySystem, audit.EntitySystem
	}
	if err := s.auditLogger.Log(nil, audit.EventAdminBreakGlassFailed, entityType, entityID, nil, map[string]any{
		"via":            "cli",
		"command":        "recover-admin",
		"email":          email,
		"failure_reason": reason,
		"hostname":       bg.Hostname,
		"os_user":        bg.OSUser,
	}); err != nil {
		log.Printf("AUDIT WRITE FAILED for admin_break_glass_failed (%s, %s): %v", email, reason, err)
	}
}
