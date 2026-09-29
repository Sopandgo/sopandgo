package mail

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

const (
	MailModeSMTP        = "smtp"
	MailModeManualLinks = "manual_links"

	MailTransportSMTP   = "smtp"
	MailTransportResend = "resend"
)

// SMTPSettingsStore persists SMTP settings in smtp_settings (single row id=1).
type SMTPSettingsStore struct {
	db  *sql.DB
	key []byte // 32 bytes from env; may be nil
}

// NewSMTPSettingsStore creates a store. key may be nil if env key is not set (saves and sends will fail until configured).
func NewSMTPSettingsStore(db *sql.DB, key []byte) *SMTPSettingsStore {
	return &SMTPSettingsStore{db: db, key: key}
}

// KeyConfigured reports whether a 32-byte encryption key is available.
func (s *SMTPSettingsStore) KeyConfigured() bool {
	return len(s.key) == 32
}

// PublicSMTPSettings is safe to return to clients (no secrets).
type PublicSMTPSettings struct {
	Host               string `json:"host"`
	Port               string `json:"port"`
	Username           string `json:"username"`
	FromAddress        string `json:"from_address"`
	MailMode           string `json:"mail_mode"`
	MailTransport      string `json:"mail_transport"`
	PasswordConfigured bool   `json:"password_configured"`
	Configured         bool   `json:"configured"`
	EncryptionKeySet   bool   `json:"encryption_key_set"`
	// Resend (optional; used when mail_transport is "resend")
	ResendFromAddress        string `json:"resend_from_address"`
	ResendConfigured         bool   `json:"resend_configured"`
	ResendAPIKeyConfigured   bool   `json:"resend_api_key_configured"`
}

// GetPublic returns current settings for the admin UI.
func (s *SMTPSettingsStore) GetPublic() (*PublicSMTPSettings, error) {
	mailMode, err := s.GetMailMode()
	if err != nil {
		return nil, err
	}
	transport, err := s.GetMailTransport()
	if err != nil {
		return nil, err
	}

	pub := &PublicSMTPSettings{
		MailMode:      mailMode,
		MailTransport: transport,
		EncryptionKeySet: s.KeyConfigured(),
	}

	row := s.db.QueryRow(`
		SELECT host, port, username, length(password_enc), from_addr
		FROM smtp_settings WHERE id = 1
	`)
	var host, port, user, from string
	var pwdLen int
	err = row.Scan(&host, &port, &user, &pwdLen, &from)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// leave SMTP fields empty
	case err != nil:
		return nil, err
	default:
		pub.Host = host
		pub.Port = port
		pub.Username = user
		pub.FromAddress = from
		pub.PasswordConfigured = pwdLen > 0
		pub.Configured = true
	}

	rrow := s.db.QueryRow(`
		SELECT from_addr, length(api_key_enc) FROM resend_settings WHERE id = 1
	`)
	var rFrom string
	var keyLen int
	err = rrow.Scan(&rFrom, &keyLen)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// optional
	case err != nil:
		return nil, err
	default:
		pub.ResendFromAddress = rFrom
		pub.ResendConfigured = true
		pub.ResendAPIKeyConfigured = keyLen > 0
	}

	return pub, nil
}

func (s *SMTPSettingsStore) GetMailMode() (string, error) {
	var mode string
	err := s.db.QueryRow(`SELECT mail_mode FROM app_settings WHERE id = 1`).Scan(&mode)
	if errors.Is(err, sql.ErrNoRows) {
		return MailModeSMTP, nil
	}
	if err != nil {
		return "", err
	}
	if mode != MailModeSMTP && mode != MailModeManualLinks {
		return MailModeSMTP, nil
	}
	return mode, nil
}

func (s *SMTPSettingsStore) SetMailMode(mode string) error {
	if mode != MailModeSMTP && mode != MailModeManualLinks {
		return ErrInvalidMailMode
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`
		INSERT INTO app_settings (id, mail_mode, updated_at)
		VALUES (1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			mail_mode = excluded.mail_mode,
			updated_at = excluded.updated_at
	`, mode, now)
	return err
}

func (s *SMTPSettingsStore) GetMailTransport() (string, error) {
	var transport string
	err := s.db.QueryRow(`SELECT mail_transport FROM app_settings WHERE id = 1`).Scan(&transport)
	if errors.Is(err, sql.ErrNoRows) {
		return MailTransportSMTP, nil
	}
	if err != nil {
		return "", err
	}
	if transport != MailTransportSMTP && transport != MailTransportResend {
		return MailTransportSMTP, nil
	}
	return transport, nil
}

func (s *SMTPSettingsStore) SetMailTransport(transport string) error {
	if transport != MailTransportSMTP && transport != MailTransportResend {
		return ErrInvalidMailTransport
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`
		UPDATE app_settings SET mail_transport = ?, updated_at = ? WHERE id = 1
	`, transport, now)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = s.db.Exec(`
			INSERT INTO app_settings (id, mail_mode, mail_transport, updated_at)
			VALUES (1, 'smtp', ?, ?)
		`, transport, now)
	}
	return err
}

// SaveResendInput persists Resend API credentials (API key encrypted at rest).
type SaveResendInput struct {
	FromAddress string
	APIKey      string // empty on update means keep existing encrypted key
}

// SaveResend upserts the single Resend row. Requires KeyConfigured. API key required when no row exists yet.
func (s *SMTPSettingsStore) SaveResend(in SaveResendInput) error {
	if !s.KeyConfigured() {
		return ErrSMTPKeyMissing
	}
	var existingKeyEnc []byte
	err := s.db.QueryRow(`SELECT api_key_enc FROM resend_settings WHERE id = 1`).Scan(&existingKeyEnc)
	hasRow := true
	if errors.Is(err, sql.ErrNoRows) {
		hasRow = false
	} else if err != nil {
		return err
	}

	keyEnc := existingKeyEnc
	if in.APIKey != "" {
		keyEnc, err = secrets.Seal(s.key, []byte(in.APIKey))
		if err != nil {
			return fmt.Errorf("encrypt resend api key: %w", err)
		}
	} else if !hasRow {
		return ErrResendAPIKeyRequired
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if hasRow {
		_, err = s.db.Exec(`
			UPDATE resend_settings SET
				api_key_enc = ?, from_addr = ?, updated_at = ?
			WHERE id = 1
		`, keyEnc, in.FromAddress, now)
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO resend_settings (id, api_key_enc, from_addr, updated_at)
		VALUES (1, ?, ?, ?)
	`, keyEnc, in.FromAddress, now)
	return err
}

func (s *SMTPSettingsStore) loadResendDecrypted() (apiKey, from string, err error) {
	if !s.KeyConfigured() {
		return "", "", ErrSMTPKeyMissing
	}
	var keyEnc []byte
	err = s.db.QueryRow(`
		SELECT api_key_enc, from_addr FROM resend_settings WHERE id = 1
	`).Scan(&keyEnc, &from)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrResendNotConfigured
	}
	if err != nil {
		return "", "", err
	}
	raw, err := secrets.Open(s.key, keyEnc)
	if err != nil {
		return "", "", fmt.Errorf("decrypt resend api key (wrong SMTP_SECRET_ENCRYPTION_KEY?): %w", err)
	}
	return string(raw), from, nil
}

// SaveSMTPInput is used to create or update SMTP settings.
type SaveSMTPInput struct {
	Host        string
	Port        string
	Username    string
	FromAddress string
	Password    string // empty on update means keep existing encrypted password
}

// Save upserts the single SMTP row. Requires KeyConfigured. Password required when no row exists yet.
func (s *SMTPSettingsStore) Save(in SaveSMTPInput) error {
	if !s.KeyConfigured() {
		return ErrSMTPKeyMissing
	}
	var existingPwdEnc []byte
	err := s.db.QueryRow(`SELECT password_enc FROM smtp_settings WHERE id = 1`).Scan(&existingPwdEnc)
	hasRow := true
	if errors.Is(err, sql.ErrNoRows) {
		hasRow = false
	} else if err != nil {
		return err
	}

	pwdEnc := existingPwdEnc
	if in.Password != "" {
		pwdEnc, err = secrets.Seal(s.key, []byte(in.Password))
		if err != nil {
			return fmt.Errorf("encrypt smtp password: %w", err)
		}
	} else if !hasRow {
		return ErrSMTPPasswordRequired
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if hasRow {
		_, err = s.db.Exec(`
			UPDATE smtp_settings SET
				host = ?, port = ?, username = ?, password_enc = ?, from_addr = ?, updated_at = ?
			WHERE id = 1
		`, in.Host, in.Port, in.Username, pwdEnc, in.FromAddress, now)
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO smtp_settings (id, host, port, username, password_enc, from_addr, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
	`, in.Host, in.Port, in.Username, pwdEnc, in.FromAddress, now)
	return err
}

// SendTestEmail sends a simple message using the active transport (SMTP or Resend).
func (s *SMTPSettingsStore) SendTestEmail(to string) error {
	transport, err := s.GetMailTransport()
	if err != nil {
		return err
	}
	subject := "SOPandGO email test"
	if transport == MailTransportResend {
		body := "This is a test message from your SOPandGO server. If you received this, Resend is configured correctly."
		return s.sendResendEmail([]string{to}, subject, body)
	}
	host, port, user, pass, from, err := s.loadDecrypted()
	if err != nil {
		return err
	}
	sender := NewSmtpSender(host, port, user, pass, from)
	body := "This is a test message from your SOPandGO server. If you received this, SMTP is configured correctly."
	return sender.Send([]string{to}, subject, body)
}

// SendOperational delivers transactional mail using the configured transport (SMTP or Resend).
func (s *SMTPSettingsStore) SendOperational(to []string, subject, body string) error {
	transport, err := s.GetMailTransport()
	if err != nil {
		return err
	}
	if transport == MailTransportResend {
		return s.sendResendEmail(to, subject, body)
	}
	host, port, user, pass, from, err := s.loadDecrypted()
	if err != nil {
		return err
	}
	sender := NewSmtpSender(host, port, user, pass, from)
	return sender.Send(to, subject, body)
}

func (s *SMTPSettingsStore) loadDecrypted() (host, port, user, pass, from string, err error) {
	if !s.KeyConfigured() {
		return "", "", "", "", "", ErrSMTPKeyMissing
	}
	var pwdEnc []byte
	err = s.db.QueryRow(`
		SELECT host, port, username, password_enc, from_addr FROM smtp_settings WHERE id = 1
	`).Scan(&host, &port, &user, &pwdEnc, &from)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", "", "", ErrSMTPNotConfigured
	}
	if err != nil {
		return "", "", "", "", "", err
	}
	raw, err := secrets.Open(s.key, pwdEnc)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("decrypt smtp password (wrong SMTP_SECRET_ENCRYPTION_KEY?): %w", err)
	}
	return host, port, user, string(raw), from, nil
}
