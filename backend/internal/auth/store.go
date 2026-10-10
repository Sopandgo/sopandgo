package auth

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// --- User ---

func createUserRecord(db audit.DBTX, id, displayName, email, passwordHash, role, createdAt string) error {
	_, err := db.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, created_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`, id, displayName, email, passwordHash, role, createdAt)
	return err
}

func applyAvatarFields(u *User, avatarPath, avatarHash sql.NullString) {
	if avatarPath.Valid && strings.TrimSpace(avatarPath.String) != "" {
		u.HasAvatar = true
		if avatarHash.Valid {
			u.AvatarContentHash = avatarHash.String
		}
	}
}

func getUserByIDRecord(db audit.DBTX, id string) (*User, error) {
	const query = `
		SELECT id, display_name, email, role_id, is_active, must_change_password, locale, theme,
			avatar_path, avatar_content_hash, created_at
		FROM users WHERE id = ?`

	var u User
	var createdAt string
	var isActive int
	var mustChange int
	var avatarPath, avatarHash sql.NullString

	err := db.QueryRow(query, id).Scan(
		&u.ID, &u.DisplayName, &u.Email, &u.Role, &isActive, &mustChange, &u.Locale, &u.Theme,
		&avatarPath, &avatarHash, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	u.IsActive = isActive == 1
	u.MustChangePassword = mustChange == 1
	applyAvatarFields(&u, avatarPath, avatarHash)
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	return &u, nil
}

func getUserByEmailRecord(db audit.DBTX, email string) (*User, string, error) {
	const query = `
		SELECT id, display_name, email, role_id, is_active, must_change_password, locale, theme,
			avatar_path, avatar_content_hash, password_hash
		FROM users WHERE email = ?`

	var u User
	var hash string
	var isActive int
	var mustChange int
	var avatarPath, avatarHash sql.NullString

	err := db.QueryRow(query, email).Scan(
		&u.ID, &u.DisplayName, &u.Email, &u.Role, &isActive, &mustChange, &u.Locale, &u.Theme,
		&avatarPath, &avatarHash, &hash,
	)
	if err != nil {
		return nil, "", err
	}

	u.IsActive = isActive == 1
	u.MustChangePassword = mustChange == 1
	applyAvatarFields(&u, avatarPath, avatarHash)
	return &u, hash, nil
}

func getUserPasswordHashRecord(db audit.DBTX, userID string) (string, error) {
	var hash string
	err := db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash)
	return hash, err
}

func getUserRoleRecord(db audit.DBTX, userID string) (string, error) {
	var role string
	err := db.QueryRow(`SELECT role_id FROM users WHERE id = ?`, userID).Scan(&role)
	return role, err
}

func getUserActiveStatusRecord(db audit.DBTX, userID string) (bool, error) {
	var isActive int
	err := db.QueryRow(`SELECT is_active FROM users WHERE id = ?`, userID).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive == 1, nil
}

func updateUserRoleRecord(db audit.DBTX, userID, newRole string) error {
	_, err := db.Exec(`UPDATE users SET role_id = ? WHERE id = ?`, newRole, userID)
	return err
}

func updateUserStatusRecord(db audit.DBTX, userID string, active int) error {
	_, err := db.Exec(`UPDATE users SET is_active = ? WHERE id = ?`, active, userID)
	return err
}

func updateUserPasswordRecord(db audit.DBTX, userID, hash string) error {
	_, err := db.Exec(`
		UPDATE users SET password_hash = ?, must_change_password = 0 WHERE id = ?
	`, hash, userID)
	return err
}

func setMustChangePasswordRecord(db audit.DBTX, userID string, mustChange bool) error {
	flag := 0
	if mustChange {
		flag = 1
	}
	_, err := db.Exec(`UPDATE users SET must_change_password = ? WHERE id = ?`, flag, userID)
	return err
}

func updateUserLocaleRecord(db audit.DBTX, userID, locale string) error {
	_, err := db.Exec(`UPDATE users SET locale = ? WHERE id = ?`, locale, userID)
	return err
}

func updateUserThemeRecord(db audit.DBTX, userID, theme string) error {
	_, err := db.Exec(`UPDATE users SET theme = ? WHERE id = ?`, theme, userID)
	return err
}

func listUsersRecord(db audit.DBTX) ([]User, error) {
	const query = `
		SELECT id, display_name, email, role_id, is_active, must_change_password, locale, theme,
			avatar_path, avatar_content_hash, created_at
		FROM users ORDER BY created_at DESC, rowid DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var createdAt string
		var isActive int
		var mustChange int
		var avatarPath, avatarHash sql.NullString
		if err := rows.Scan(
			&u.ID, &u.DisplayName, &u.Email, &u.Role, &isActive, &mustChange, &u.Locale, &u.Theme,
			&avatarPath, &avatarHash, &createdAt,
		); err != nil {
			return nil, err
		}
		u.IsActive = isActive == 1
		u.MustChangePassword = mustChange == 1
		applyAvatarFields(&u, avatarPath, avatarHash)
		u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		users = append(users, u)
	}
	return users, nil
}

func getUserAvatarMetaRecord(db audit.DBTX, userID string) (path, hash string, err error) {
	var pathNS, hashNS sql.NullString
	err = db.QueryRow(`
		SELECT avatar_path, avatar_content_hash FROM users WHERE id = ?
	`, userID).Scan(&pathNS, &hashNS)
	if err != nil {
		return "", "", err
	}
	if pathNS.Valid {
		path = pathNS.String
	}
	if hashNS.Valid {
		hash = hashNS.String
	}
	return path, hash, nil
}

func setUserAvatarRecord(db audit.DBTX, userID, path, hash string) error {
	_, err := db.Exec(`
		UPDATE users SET avatar_path = ?, avatar_content_hash = ? WHERE id = ?
	`, path, hash, userID)
	return err
}

func clearUserAvatarRecord(db audit.DBTX, userID string) error {
	_, err := db.Exec(`
		UPDATE users SET avatar_path = NULL, avatar_content_hash = NULL WHERE id = ?
	`, userID)
	return err
}

// --- Session ---

// getActiveTokenOwnerRecord finds the user ID for a token, but ONLY if it is currently active.
// This prevents logging "User Logout" events for sessions that were already revoked.
func getActiveTokenOwnerRecord(db audit.DBTX, tokenID string) (string, error) {
	var userID string
	err := db.QueryRow(`
        SELECT user_id 
        FROM refresh_tokens 
        WHERE token_id = ? AND is_active = 1
    `, tokenID).Scan(&userID)

	return userID, err
}

func cleanupExpiredTokensRecord(db audit.DBTX) error {
	_, err := db.Exec("DELETE FROM refresh_tokens WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func insertRefreshTokenRecord(db audit.DBTX, tokenID, userID string, expiresAt, createdAt time.Time) error {
	_, err := db.Exec(`
        INSERT INTO refresh_tokens (token_id, user_id, expires_at, created_at, is_active)
        VALUES (?, ?, ?, ?, 1)
    `, tokenID, userID, expiresAt, createdAt)
	return err
}

func listActiveSessionsRecord(db audit.DBTX) ([]Session, error) {
	// token_id is the refresh secret. Do not select it into the admin list.
	const query = `
		SELECT rt.user_id, u.email, u.display_name, rt.created_at, rt.expires_at
		FROM refresh_tokens rt
		JOIN users u ON rt.user_id = u.id
		WHERE rt.is_active = 1 AND rt.expires_at > CURRENT_TIMESTAMP
		ORDER BY rt.created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var created, expires string
		if err := rows.Scan(&s.UserID, &s.UserEmail, &s.UserName, &created, &expires); err != nil {
			return nil, err
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		s.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// ValidateRefreshTokenRecord checks if a token is valid, active, and belongs to an active user.
// It returns the user's ID and Role if successful.
func validateRefreshTokenRecord(db audit.DBTX, tokenID string) (string, string, error) {
	var userID, role string
	var tokenActive, userActive int

	// Join the token with the user to ensure BOTH are active
	err := db.QueryRow(`
        SELECT rt.user_id, u.role_id, rt.is_active, u.is_active
        FROM refresh_tokens rt
        JOIN users u ON rt.user_id = u.id
        WHERE rt.token_id = ? AND rt.expires_at > CURRENT_TIMESTAMP
    `, tokenID).Scan(&userID, &role, &tokenActive, &userActive)

	if err != nil {
		return "", "", err
	}

	if tokenActive == 0 {
		return "", "", fmt.Errorf("session is revoked")
	}
	if userActive == 0 {
		return "", "", fmt.Errorf("user account is disabled")
	}

	return userID, role, nil
}

// revokes a single active session
func revokeRefreshTokenRecord(db audit.DBTX, tokenID string) error {
	_, err := db.Exec("UPDATE refresh_tokens SET is_active = 0 WHERE token_id = ?", tokenID)
	return err
}

// revokes every active session for a specific person
func revokeUserRefreshTokensRecord(db audit.DBTX, userID string) error {
	_, err := db.Exec("UPDATE refresh_tokens SET is_active = 0 WHERE user_id = ?", userID)
	return err
}

// revokes every active session for a person except the one they are using.
// Returns how many sessions were revoked.
func revokeOtherUserRefreshTokensRecord(db audit.DBTX, userID, keepTokenID string) (int64, error) {
	res, err := db.Exec(
		"UPDATE refresh_tokens SET is_active = 0 WHERE user_id = ? AND token_id != ? AND is_active = 1",
		userID, keepTokenID,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// globally revokes every active session
func revokeAllRefreshTokensRecord(db audit.DBTX) error {
	_, err := db.Exec("UPDATE refresh_tokens SET is_active = 0")
	return err
}

// --- Admin ---
// countAdminsRecord checks how many administrators exist in the database.
func countAdminsRecord(db audit.DBTX) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE role_id = ?`, RoleAdmin).Scan(&count)
	return count, err
}

// --- Password Reset ---

// saveResetTokenRecord invalidates any existing token for the user and saves the new one.
// We use "INSERT OR REPLACE" to handle the "Single Token Per User" rule atomically.
func saveResetTokenRecord(db audit.DBTX, userID, tokenHash string, expiresAt time.Time) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, userID, tokenHash, expiresAt)
	return err
}

// getResetTokenRecord retrieves the user ID and expiration for a given token hash.
func getResetTokenRecord(db audit.DBTX, tokenHash string) (string, time.Time, error) {
	var userID string
	var expiresStr string

	err := db.QueryRow(`
		SELECT user_id, expires_at 
		FROM password_reset_tokens 
		WHERE token_hash = ?
	`, tokenHash).Scan(&userID, &expiresStr)

	if err != nil {
		return "", time.Time{}, err
	}

	// Parse the SQLite datetime string back into a Go time object
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresStr)
	if err != nil {
		// Fallback for some SQLite drivers that might omit the 'T' or timezone
		// standard format: "2006-01-02 15:04:05"
		expiresAt, err = time.Parse("2006-01-02 15:04:05", expiresStr)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("failed to parse token expiry time: %w", err)
		}
	}

	return userID, expiresAt, nil
}

// deleteResetTokenRecord removes the token for a specific user.
// Used after a successful password reset or for manual invalidation.
func deleteResetTokenRecord(db audit.DBTX, userID string) error {
	_, err := db.Exec(`DELETE FROM password_reset_tokens WHERE user_id = ?`, userID)
	return err
}
