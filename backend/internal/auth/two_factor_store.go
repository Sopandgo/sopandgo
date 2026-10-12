package auth

import (
	"database/sql"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// --- Two-factor ---

type twoFactorRow struct {
	secretEnc    []byte // confirmed authenticator; nil until the first confirm
	pendingEnc   []byte // authenticator waiting for its first code
	confirmed    bool
	lastUsedStep int64
}

func getTwoFactorRecord(db audit.DBTX, userID string) (*twoFactorRow, error) {
	var row twoFactorRow
	var confirmedAt sql.NullString
	err := db.QueryRow(`
		SELECT totp_secret_enc, pending_secret_enc, confirmed_at, last_used_step
		FROM user_two_factor WHERE user_id = ?
	`, userID).Scan(&row.secretEnc, &row.pendingEnc, &confirmedAt, &row.lastUsedStep)
	if err != nil {
		return nil, err
	}
	row.confirmed = confirmedAt.Valid && row.secretEnc != nil
	return &row, nil
}

func savePendingTwoFactorRecord(db audit.DBTX, userID string, pendingEnc []byte, now time.Time) error {
	_, err := db.Exec(`
		INSERT INTO user_two_factor (user_id, pending_secret_enc, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET pending_secret_enc = excluded.pending_secret_enc
	`, userID, pendingEnc, now.UTC().Format(time.RFC3339Nano))
	return err
}

// confirmTwoFactorRecord promotes the pending secret to the active one.
func confirmTwoFactorRecord(db audit.DBTX, userID string, step int64, now time.Time) error {
	_, err := db.Exec(`
		UPDATE user_two_factor
		SET totp_secret_enc = pending_secret_enc, pending_secret_enc = NULL,
			confirmed_at = ?, last_used_step = ?
		WHERE user_id = ? AND pending_secret_enc IS NOT NULL
	`, now.UTC().Format(time.RFC3339Nano), step, userID)
	return err
}

// advanceTwoFactorStepRecord records a used TOTP step. It changes nothing when
// the step is not newer than the last one, so a replayed code is rejected even
// if two requests race.
func advanceTwoFactorStepRecord(db audit.DBTX, userID string, step int64) (bool, error) {
	res, err := db.Exec(`
		UPDATE user_two_factor SET last_used_step = ?
		WHERE user_id = ? AND confirmed_at IS NOT NULL AND last_used_step < ?
	`, step, userID, step)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// deleteTwoFactorRecords removes every second-factor row for a user and
// reports which kinds existed.
func deleteTwoFactorRecords(db audit.DBTX, userID string) (totp, recovery, email int64, err error) {
	res, err := db.Exec(`DELETE FROM user_two_factor WHERE user_id = ?`, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	totp, _ = res.RowsAffected()
	recovery, err = deleteRecoveryCodesRecord(db, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	res, err = db.Exec(`DELETE FROM two_factor_email_codes WHERE user_id = ?`, userID)
	if err != nil {
		return 0, 0, 0, err
	}
	email, _ = res.RowsAffected()
	return totp, recovery, email, nil
}

func getRequireTwoFactorRecord(db audit.DBTX) (bool, error) {
	var v int
	err := db.QueryRow(`SELECT require_two_factor FROM app_settings WHERE id = 1`).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return v == 1, err
}

func setRequireTwoFactorRecord(db audit.DBTX, required bool, now time.Time) error {
	v := 0
	if required {
		v = 1
	}
	_, err := db.Exec(`
		INSERT INTO app_settings (id, require_two_factor, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET require_two_factor = excluded.require_two_factor,
			updated_at = excluded.updated_at
	`, v, now.UTC().Format(time.RFC3339Nano))
	return err
}

// --- Recovery codes ---

func deleteRecoveryCodesRecord(db audit.DBTX, userID string) (int64, error) {
	res, err := db.Exec(`DELETE FROM two_factor_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func insertRecoveryCodeRecord(db audit.DBTX, userID, codeHash string, now time.Time) error {
	_, err := db.Exec(`
		INSERT INTO two_factor_recovery_codes (user_id, code_hash, created_at) VALUES (?, ?, ?)
	`, userID, codeHash, now.UTC().Format(time.RFC3339Nano))
	return err
}

// useRecoveryCodeRecord burns a matching code. It reports false when no code matched.
func useRecoveryCodeRecord(db audit.DBTX, userID, codeHash string) (bool, error) {
	res, err := db.Exec(`
		DELETE FROM two_factor_recovery_codes WHERE user_id = ? AND code_hash = ?
	`, userID, codeHash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func countRecoveryCodesRecord(db audit.DBTX, userID string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM two_factor_recovery_codes WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// --- Email codes ---

type emailCodeRow struct {
	codeHash  string
	expiresAt time.Time
	attempts  int
	createdAt time.Time
}

func getEmailCodeRecord(db audit.DBTX, userID string) (*emailCodeRow, error) {
	var row emailCodeRow
	var expires, created string
	err := db.QueryRow(`
		SELECT code_hash, expires_at, attempts, created_at FROM two_factor_email_codes WHERE user_id = ?
	`, userID).Scan(&row.codeHash, &expires, &row.attempts, &created)
	if err != nil {
		return nil, err
	}
	row.expiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	row.createdAt, _ = time.Parse(time.RFC3339Nano, created)
	return &row, nil
}

func saveEmailCodeRecord(db audit.DBTX, userID, codeHash string, expiresAt, now time.Time) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO two_factor_email_codes (user_id, code_hash, expires_at, attempts, created_at)
		VALUES (?, ?, ?, 0, ?)
	`, userID, codeHash, expiresAt.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano))
	return err
}

func incrementEmailCodeAttemptsRecord(db audit.DBTX, userID string) error {
	_, err := db.Exec(`UPDATE two_factor_email_codes SET attempts = attempts + 1 WHERE user_id = ?`, userID)
	return err
}

func deleteEmailCodeRecord(db audit.DBTX, userID string) error {
	_, err := db.Exec(`DELETE FROM two_factor_email_codes WHERE user_id = ?`, userID)
	return err
}

// --- Break-glass ---

func reactivateUserRecord(db audit.DBTX, userID string) (bool, error) {
	res, err := db.Exec(`UPDATE users SET is_active = 1 WHERE id = ? AND is_active = 0`, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func countActiveUserSessionsRecord(db audit.DBTX, userID string) (int64, error) {
	var n int64
	err := db.QueryRow(`
		SELECT COUNT(*) FROM refresh_tokens WHERE user_id = ? AND is_active = 1
	`, userID).Scan(&n)
	return n, err
}

func listAdminsRecord(db audit.DBTX) ([]User, error) {
	users, err := listUsersRecord(db)
	if err != nil {
		return nil, err
	}
	admins := make([]User, 0, len(users))
	for _, u := range users {
		if u.Role == RoleAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}
