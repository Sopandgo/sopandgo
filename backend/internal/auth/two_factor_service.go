package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

var (
	ErrNoEncryptionKey         = errors.New("SECRET_ENCRYPTION_KEY is not configured")
	ErrTwoFactorNotEnabled     = errors.New("two-factor authentication is not enabled")
	ErrNoPendingTwoFactor      = errors.New("no authenticator is waiting for confirmation")
	ErrInvalidSecondFactor     = errors.New("invalid verification code")
	ErrTwoFactorRequired       = errors.New("the organization requires two-factor authentication")
	ErrTwoFactorUnavailable    = errors.New("the stored authenticator secret cannot be read")
	ErrCannotResetAdmin        = errors.New("admin accounts are recovered with recovery codes or the server's recover-admin command")
	ErrRecoveryCodesAdminOnly  = errors.New("recovery codes are only available to admins")
	ErrEmailCodeThrottled      = errors.New("a code was sent recently; wait before requesting another")
	ErrActorNeedsTwoFactor     = errors.New("turn on two-factor authentication for your own account first")
	ErrUnknownSecondFactorType = errors.New("unknown second factor method")
)

// Second-factor methods accepted at sign-in.
const (
	SecondFactorTOTP         = "totp"
	SecondFactorEmail        = "email"
	SecondFactorRecoveryCode = "recovery_code"
)

const (
	totpIssuer           = "SOPandGO"
	totpPeriod           = 30
	totpSkewSteps        = 1
	recoveryCodeCount    = 10
	emailCodeTTL         = 10 * time.Minute
	emailCodeResendGap   = 60 * time.Second
	emailCodeMaxAttempts = 5
	qrCodeSize           = 240
)

// TOTPSetup is what the user needs to add the authenticator to an app.
// The secret is returned once and never again.
type TOTPSetup struct {
	Secret    string `json:"secret"`
	URI       string `json:"otpauth_uri"`
	QRCodePNG string `json:"qr_code_data_url"`
}

// TwoFactorStatus describes one account's second factor for the settings page.
type TwoFactorStatus struct {
	Enabled                 bool `json:"enabled"`
	PendingSetup            bool `json:"pending_setup"`
	Required                bool `json:"required"`
	RecoveryCodesRemaining  int  `json:"recovery_codes_remaining"`
	RecoveryCodesAvailable  bool `json:"recovery_codes_available"`
	EncryptionKeyConfigured bool `json:"encryption_key_configured"`
}

func (s *Service) sealSecret(secret string) ([]byte, error) {
	if !s.EncryptionKeyConfigured() {
		return nil, ErrNoEncryptionKey
	}
	return secrets.Seal(s.encKey, []byte(secret))
}

func (s *Service) openSecret(blob []byte) (string, error) {
	if !s.EncryptionKeyConfigured() {
		return "", ErrTwoFactorUnavailable
	}
	plain, err := secrets.Open(s.encKey, blob)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTwoFactorUnavailable, err)
	}
	return string(plain), nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// normalizeRecoveryCode accepts codes with or without the dash, in any case.
func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	return strings.ReplaceAll(code, " ", "")
}

func normalizeNumericCode(code string) string {
	return strings.ReplaceAll(strings.TrimSpace(code), " ", "")
}

// matchTOTP returns the time step the code belongs to, allowing one step of
// clock drift either way.
func matchTOTP(secret, code string, now time.Time) (int64, bool) {
	code = normalizeNumericCode(code)
	if len(code) != 6 {
		return 0, false
	}
	opts := totp.ValidateOpts{Period: totpPeriod, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
	for skew := -totpSkewSteps; skew <= totpSkewSteps; skew++ {
		at := now.Add(time.Duration(skew*totpPeriod) * time.Second)
		want, err := totp.GenerateCodeCustom(secret, at, opts)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return at.Unix() / totpPeriod, true
		}
	}
	return 0, false
}

func randomString(alphabet string, n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out), nil
}

// recoveryAlphabet leaves out 0/O and 1/I so codes can be read off paper.
const recoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newRecoveryCode() (string, error) {
	raw, err := randomString(recoveryAlphabet, 10)
	if err != nil {
		return "", err
	}
	return raw[:5] + "-" + raw[5:], nil
}

// issueRecoveryCodes replaces a user's recovery codes inside tx and returns the new ones.
func issueRecoveryCodes(tx audit.DBTX, userID string, now time.Time) ([]string, error) {
	if _, err := deleteRecoveryCodesRecord(tx, userID); err != nil {
		return nil, err
	}
	codes := make([]string, 0, recoveryCodeCount)
	for len(codes) < recoveryCodeCount {
		code, err := newRecoveryCode()
		if err != nil {
			return nil, err
		}
		if err := insertRecoveryCodeRecord(tx, userID, hashCode(normalizeRecoveryCode(code)), now); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func (s *Service) verifyOwnPassword(userID, currentPassword string) error {
	hash, err := getUserPasswordHashRecord(s.db, userID)
	if err != nil {
		return fmt.Errorf("failed to load password: %w", err)
	}
	if err := VerifyPassword(currentPassword, hash); err != nil {
		return ErrWrongCurrentPassword
	}
	return nil
}

// TwoFactorStatus reports the user's second factor and the organization rule.
func (s *Service) TwoFactorStatus(userID string) (*TwoFactorStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, err := getUserByIDRecord(s.db, userID)
	if err != nil {
		return nil, err
	}
	required, err := getRequireTwoFactorRecord(s.db)
	if err != nil {
		return nil, err
	}
	st := &TwoFactorStatus{
		Enabled:                 user.TwoFactorEnabled,
		Required:                required,
		RecoveryCodesAvailable:  user.Role == RoleAdmin,
		EncryptionKeyConfigured: s.EncryptionKeyConfigured(),
	}
	row, err := getTwoFactorRecord(s.db, userID)
	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return nil, err
	default:
		st.PendingSetup = row.pendingEnc != nil
	}
	if st.RecoveryCodesAvailable {
		if st.RecoveryCodesRemaining, err = countRecoveryCodesRecord(s.db, userID); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// RequireTwoFactor reports whether the organization requires 2FA.
func (s *Service) RequireTwoFactor() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getRequireTwoFactorRecord(s.db)
}

// BeginTOTPSetup creates a new authenticator secret and keeps it pending until
// ConfirmTOTPSetup sees a valid code. An existing authenticator stays active
// until then, so a replacement cannot lock the user out halfway.
func (s *Service) BeginTOTPSetup(userID, currentPassword string) (*TOTPSetup, error) {
	if !s.EncryptionKeyConfigured() {
		return nil, ErrNoEncryptionKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.verifyOwnPassword(userID, currentPassword); err != nil {
		return nil, err
	}
	user, err := getUserByIDRecord(s.db, userID)
	if err != nil {
		return nil, err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: user.Email,
		Period:      totpPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate authenticator secret: %w", err)
	}
	sealed, err := s.sealSecret(key.Secret())
	if err != nil {
		return nil, err
	}

	img, err := key.Image(qrCodeSize, qrCodeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to render QR code: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	if err := savePendingTwoFactorRecord(s.db, userID, sealed, time.Now()); err != nil {
		return nil, fmt.Errorf("failed to store authenticator: %w", err)
	}

	return &TOTPSetup{
		Secret:    key.Secret(),
		URI:       key.URL(),
		QRCodePNG: "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// ConfirmTOTPSetup activates the pending authenticator once the user proves it
// works. When keepSessionToken is one of the user's active sessions, every
// other session is signed out. Admins get a fresh set of recovery codes,
// returned once.
func (s *Service) ConfirmTOTPSetup(userID, code, keepSessionToken string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, err := getTwoFactorRecord(s.db, userID)
	if err == sql.ErrNoRows || (err == nil && row.pendingEnc == nil) {
		return nil, ErrNoPendingTwoFactor
	}
	if err != nil {
		return nil, err
	}
	secret, err := s.openSecret(row.pendingEnc)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	step, ok := matchTOTP(secret, code, now)
	if !ok {
		return nil, ErrInvalidSecondFactor
	}
	user, err := getUserByIDRecord(s.db, userID)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	replaced := row.confirmed
	if err := confirmTwoFactorRecord(tx, userID, step, now); err != nil {
		return nil, fmt.Errorf("failed to confirm authenticator: %w", err)
	}

	var codes []string
	if user.Role == RoleAdmin {
		if codes, err = issueRecoveryCodes(tx, userID, now); err != nil {
			return nil, fmt.Errorf("failed to issue recovery codes: %w", err)
		}
	}

	var revoked int64
	if keepSessionToken != "" {
		if owner, err := getActiveTokenOwnerRecord(tx, keepSessionToken); err == nil && owner == userID {
			if revoked, err = revokeOtherUserRefreshTokensRecord(tx, userID, keepSessionToken); err != nil {
				return nil, fmt.Errorf("failed to revoke other sessions: %w", err)
			}
		}
	}

	if err := s.auditLogger.Log(tx, audit.EventTwoFactorEnabled, audit.EntityUser, userID, &userID, map[string]any{
		"method":                 SecondFactorTOTP,
		"replaced_authenticator": replaced,
		"recovery_codes_issued":  len(codes),
		"other_sessions_revoked": revoked,
	}); err != nil {
		return nil, err
	}
	return codes, tx.Commit()
}

// DisableTwoFactor turns 2FA off for the signed-in user. It needs the current
// password, like a password change, and is refused while the organization
// requires 2FA. It also cancels an unconfirmed setup.
func (s *Service) DisableTwoFactor(userID, currentPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.verifyOwnPassword(userID, currentPassword); err != nil {
		return err
	}
	row, err := getTwoFactorRecord(s.db, userID)
	if err == sql.ErrNoRows {
		return ErrTwoFactorNotEnabled
	}
	if err != nil {
		return err
	}
	if row.confirmed {
		required, err := getRequireTwoFactorRecord(s.db)
		if err != nil {
			return err
		}
		if required {
			return ErrTwoFactorRequired
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, _, _, err := deleteTwoFactorRecords(tx, userID); err != nil {
		return fmt.Errorf("failed to remove second factor: %w", err)
	}
	if row.confirmed {
		if err := s.auditLogger.Log(tx, audit.EventTwoFactorDisabled, audit.EntityUser, userID, &userID, map[string]any{
			"method": SecondFactorTOTP,
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RegenerateRecoveryCodes replaces an admin's recovery codes. It needs the
// current password and a confirmed authenticator.
func (s *Service) RegenerateRecoveryCodes(userID, currentPassword string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.verifyOwnPassword(userID, currentPassword); err != nil {
		return nil, err
	}
	user, err := getUserByIDRecord(s.db, userID)
	if err != nil {
		return nil, err
	}
	if user.Role != RoleAdmin {
		return nil, ErrRecoveryCodesAdminOnly
	}
	if !user.TwoFactorEnabled {
		return nil, ErrTwoFactorNotEnabled
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	codes, err := issueRecoveryCodes(tx, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to issue recovery codes: %w", err)
	}
	if err := s.auditLogger.Log(tx, audit.EventTwoFactorRecoveryCodesIssued, audit.EntityUser, userID, &userID, map[string]any{
		"count": len(codes),
	}); err != nil {
		return nil, err
	}
	return codes, tx.Commit()
}

// SecondFactorMethods lists what the user can use at the sign-in challenge.
// Email is added by the caller, because only it knows whether mail is on.
func (s *Service) SecondFactorMethods(user *User) ([]string, error) {
	methods := []string{SecondFactorTOTP}
	if user.Role == RoleAdmin {
		s.mu.RLock()
		n, err := countRecoveryCodesRecord(s.db, user.ID)
		s.mu.RUnlock()
		if err != nil {
			return nil, err
		}
		if n > 0 {
			methods = append(methods, SecondFactorRecoveryCode)
		}
	}
	return methods, nil
}

// VerifySecondFactor checks one sign-in code. A TOTP step, email code, or
// recovery code is accepted once. For a recovery code it also returns how many
// codes the admin has left.
func (s *Service) VerifySecondFactor(userID, method, code string) (remainingRecoveryCodes int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch method {
	case SecondFactorTOTP:
		return 0, s.verifyTOTPLocked(userID, code)
	case SecondFactorEmail:
		return 0, s.verifyEmailCodeLocked(userID, code)
	case SecondFactorRecoveryCode:
		return s.useRecoveryCodeLocked(userID, code)
	default:
		return 0, ErrUnknownSecondFactorType
	}
}

func (s *Service) verifyTOTPLocked(userID, code string) error {
	row, err := getTwoFactorRecord(s.db, userID)
	if err == sql.ErrNoRows || (err == nil && !row.confirmed) {
		return ErrTwoFactorNotEnabled
	}
	if err != nil {
		return err
	}
	secret, err := s.openSecret(row.secretEnc)
	if err != nil {
		return err
	}
	step, ok := matchTOTP(secret, code, time.Now())
	if !ok || step <= row.lastUsedStep {
		return ErrInvalidSecondFactor
	}
	advanced, err := advanceTwoFactorStepRecord(s.db, userID, step)
	if err != nil {
		return err
	}
	if !advanced {
		return ErrInvalidSecondFactor
	}
	return nil
}

func (s *Service) verifyEmailCodeLocked(userID, code string) error {
	row, err := getEmailCodeRecord(s.db, userID)
	if err == sql.ErrNoRows {
		return ErrInvalidSecondFactor
	}
	if err != nil {
		return err
	}
	if time.Now().After(row.expiresAt) || row.attempts >= emailCodeMaxAttempts {
		_ = deleteEmailCodeRecord(s.db, userID)
		return ErrInvalidSecondFactor
	}
	if subtle.ConstantTimeCompare([]byte(hashCode(normalizeNumericCode(code))), []byte(row.codeHash)) != 1 {
		if row.attempts+1 >= emailCodeMaxAttempts {
			_ = deleteEmailCodeRecord(s.db, userID)
		} else {
			_ = incrementEmailCodeAttemptsRecord(s.db, userID)
		}
		return ErrInvalidSecondFactor
	}
	return deleteEmailCodeRecord(s.db, userID)
}

func (s *Service) useRecoveryCodeLocked(userID, code string) (int, error) {
	role, err := getUserRoleRecord(s.db, userID)
	if err != nil {
		return 0, err
	}
	if role != RoleAdmin {
		return 0, ErrRecoveryCodesAdminOnly
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	used, err := useRecoveryCodeRecord(tx, userID, hashCode(normalizeRecoveryCode(code)))
	if err != nil {
		return 0, err
	}
	if !used {
		return 0, ErrInvalidSecondFactor
	}
	remaining, err := countRecoveryCodesRecord(tx, userID)
	if err != nil {
		return 0, err
	}
	if err := s.auditLogger.Log(tx, audit.EventTwoFactorRecoveryCodeUsed, audit.EntityUser, userID, &userID, map[string]any{
		"remaining": remaining,
	}); err != nil {
		return 0, err
	}
	return remaining, tx.Commit()
}

// IssueEmailCode creates a 6-digit sign-in code for a user with 2FA on. It
// replaces any earlier code and refuses to send more than one per minute.
func (s *Service) IssueEmailCode(userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, err := getTwoFactorRecord(s.db, userID)
	if err == sql.ErrNoRows || (err == nil && !row.confirmed) {
		return "", ErrTwoFactorNotEnabled
	}
	if err != nil {
		return "", err
	}
	now := time.Now()
	if prev, err := getEmailCodeRecord(s.db, userID); err == nil && now.Sub(prev.createdAt) < emailCodeResendGap {
		return "", ErrEmailCodeThrottled
	}

	code, err := randomString("0123456789", 6)
	if err != nil {
		return "", err
	}
	if err := saveEmailCodeRecord(s.db, userID, hashCode(code), now.Add(emailCodeTTL), now); err != nil {
		return "", err
	}
	return code, nil
}

// ResetTwoFactor is the admin action from /admin/users. It removes every
// second factor of a non-admin account and signs it out. Admin accounts are
// refused: they recover with recovery codes or the recover-admin command.
func (s *Service) ResetTwoFactor(targetUserID string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, err := getUserRoleRecord(s.db, targetUserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if role == RoleAdmin {
		return ErrCannotResetAdmin
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	totpRows, _, emailRows, err := deleteTwoFactorRecords(tx, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to remove second factor: %w", err)
	}
	if err := revokeUserRefreshTokensRecord(tx, targetUserID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	if err := s.auditLogger.Log(tx, audit.EventTwoFactorReset, audit.EntityUser, targetUserID, actorUserID, map[string]any{
		"had_authenticator": totpRows > 0,
		"had_email_code":    emailRows > 0,
		"sessions_revoked":  true,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// SetRequireTwoFactor switches the organization-wide requirement. Turning it
// on needs the encryption key and an admin who already uses 2FA, so the admin
// cannot lock themselves out by flipping it.
func (s *Service) SetRequireTwoFactor(required bool, actorUserID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := getRequireTwoFactorRecord(s.db)
	if err != nil {
		return err
	}
	if current == required {
		return nil
	}
	if required {
		if !s.EncryptionKeyConfigured() {
			return ErrNoEncryptionKey
		}
		actor, err := getUserByIDRecord(s.db, actorUserID)
		if err != nil {
			return err
		}
		if !actor.TwoFactorEnabled {
			return ErrActorNeedsTwoFactor
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setRequireTwoFactorRecord(tx, required, time.Now()); err != nil {
		return err
	}
	if err := s.auditLogger.Log(tx, audit.EventTwoFactorRequirementUpdated, audit.EntitySystem, audit.EntitySystem, &actorUserID, map[string]any{
		"from": current,
		"to":   required,
	}); err != nil {
		return err
	}
	return tx.Commit()
}
