package mail

import "errors"

var (
	// ErrSMTPNotConfigured is returned when no SMTP row exists or required fields are missing.
	ErrSMTPNotConfigured = errors.New("smtp is not configured")

	// ErrSMTPKeyMissing is returned when saving or sending requires SMTP_SECRET_ENCRYPTION_KEY but it is unset.
	ErrSMTPKeyMissing = errors.New("SMTP_SECRET_ENCRYPTION_KEY is not set or invalid")

	// ErrSMTPPasswordRequired is returned on first save without a password.
	ErrSMTPPasswordRequired = errors.New("smtp password is required on first setup")

	// ErrInvalidMailMode is returned when an unsupported mail mode is provided.
	ErrInvalidMailMode = errors.New("mail mode must be either \"smtp\" or \"manual_links\"")

	// ErrInvalidMailTransport is returned when an unsupported outbound transport is provided.
	ErrInvalidMailTransport = errors.New("mail transport must be either \"smtp\" or \"resend\"")

	// ErrResendNotConfigured is returned when Resend is selected but settings are missing.
	ErrResendNotConfigured = errors.New("resend is not configured")

	// ErrResendAPIKeyRequired is returned on first Resend save without an API key.
	ErrResendAPIKeyRequired = errors.New("resend API key is required on first setup")
)
