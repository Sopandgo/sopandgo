package notify

import "errors"

var (
	// ErrKeyMissing is returned when saving or sending requires SMTP_SECRET_ENCRYPTION_KEY but it is unset.
	ErrKeyMissing = errors.New("SMTP_SECRET_ENCRYPTION_KEY is not set or invalid")

	// ErrNotConfigured is returned when a channel is enabled but credentials are missing.
	ErrNotConfigured = errors.New("integration channel is not configured")

	// ErrSlackWebhookRequired is returned on first Slack save without a webhook URL.
	ErrSlackWebhookRequired = errors.New("slack webhook URL is required on first setup")

	// ErrGotifyTokenRequired is returned on first Gotify save without a token.
	ErrGotifyTokenRequired = errors.New("gotify token is required on first setup")

	// ErrWebhookURLRequired is returned when enabling or saving a generic webhook without a URL.
	ErrWebhookURLRequired = errors.New("webhook URL is required")

	// ErrInvalidEvent is returned when an unknown notification event type is provided.
	ErrInvalidEvent = errors.New("invalid notification event type")
)
