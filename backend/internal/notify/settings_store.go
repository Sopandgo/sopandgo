package notify

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

// SettingsStore persists outbound integration settings (single row id=1).
type SettingsStore struct {
	db  *sql.DB
	key []byte // 32 bytes from env; may be nil
}

// NewSettingsStore creates a store. key may be nil if env key is not set.
func NewSettingsStore(db *sql.DB, key []byte) *SettingsStore {
	return &SettingsStore{db: db, key: key}
}

// KeyConfigured reports whether a 32-byte encryption key is available.
func (s *SettingsStore) KeyConfigured() bool {
	return len(s.key) == 32
}

// PublicChannelSettings is safe to return to clients (no secrets).
type PublicChannelSettings struct {
	Enabled             bool     `json:"enabled"`
	Events              []string `json:"events"`
	Configured          bool     `json:"configured"`
	SecretConfigured    bool     `json:"secret_configured"`
	URL                 string   `json:"url,omitempty"` // Gotify/webhook base URL (non-secret)
}

// PublicSettings is the admin UI DTO for all channels.
type PublicSettings struct {
	EncryptionKeySet bool                  `json:"encryption_key_set"`
	Slack            PublicChannelSettings `json:"slack"`
	Gotify           PublicChannelSettings `json:"gotify"`
	Webhook          PublicChannelSettings `json:"webhook"`
	KnownEvents      []string              `json:"known_events"`
}

type rowData struct {
	slackEnabled     int
	slackURLEnc      []byte
	slackEvents      string
	gotifyEnabled    int
	gotifyURL        string
	gotifyTokenEnc   []byte
	gotifyEvents     string
	webhookEnabled   int
	webhookURL       string
	webhookBearerEnc []byte
	webhookEvents    string
}

func (s *SettingsStore) loadRow() (*rowData, error) {
	row := s.db.QueryRow(`
		SELECT
			slack_enabled, slack_webhook_url_enc, slack_events,
			gotify_enabled, gotify_url, gotify_token_enc, gotify_events,
			webhook_enabled, webhook_url, webhook_bearer_enc, webhook_events
		FROM integration_settings WHERE id = 1
	`)
	var d rowData
	err := row.Scan(
		&d.slackEnabled, &d.slackURLEnc, &d.slackEvents,
		&d.gotifyEnabled, &d.gotifyURL, &d.gotifyTokenEnc, &d.gotifyEvents,
		&d.webhookEnabled, &d.webhookURL, &d.webhookBearerEnc, &d.webhookEvents,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func emptyPublic() *PublicSettings {
	return &PublicSettings{
		Slack: PublicChannelSettings{
			Events: splitEventsOrDefault(DefaultSOPEvents),
		},
		Gotify: PublicChannelSettings{
			Events: splitEventsOrDefault(DefaultAllEvents),
		},
		Webhook: PublicChannelSettings{
			Events: splitEventsOrDefault(DefaultAllEvents),
		},
		KnownEvents: append([]string(nil), KnownEvents...),
	}
}

func splitEventsOrDefault(csv string) []string {
	events, err := ParseEventsCSV(csv)
	if err != nil || len(events) == 0 {
		events, _ = ParseEventsCSV(DefaultSOPEvents)
	}
	return events
}

// GetPublic returns current settings for the admin UI.
func (s *SettingsStore) GetPublic() (*PublicSettings, error) {
	out := emptyPublic()
	out.EncryptionKeySet = s.KeyConfigured()

	d, err := s.loadRow()
	if err != nil {
		return nil, err
	}
	if d == nil {
		return out, nil
	}

	out.Slack.Enabled = d.slackEnabled != 0
	out.Slack.Events = splitEventsOrDefault(d.slackEvents)
	out.Slack.SecretConfigured = len(d.slackURLEnc) > 0
	out.Slack.Configured = out.Slack.SecretConfigured

	out.Gotify.Enabled = d.gotifyEnabled != 0
	out.Gotify.URL = d.gotifyURL
	out.Gotify.Events = splitEventsOrDefault(d.gotifyEvents)
	out.Gotify.SecretConfigured = len(d.gotifyTokenEnc) > 0
	out.Gotify.Configured = d.gotifyURL != "" && out.Gotify.SecretConfigured

	out.Webhook.Enabled = d.webhookEnabled != 0
	out.Webhook.URL = d.webhookURL
	out.Webhook.Events = splitEventsOrDefault(d.webhookEvents)
	out.Webhook.SecretConfigured = len(d.webhookBearerEnc) > 0
	out.Webhook.Configured = d.webhookURL != ""

	return out, nil
}

// SaveSlackInput updates Slack Incoming Webhook settings.
type SaveSlackInput struct {
	Enabled    bool
	WebhookURL string // empty on update keeps existing ciphertext
	Events     []string
}

// SaveGotifyInput updates Gotify settings.
type SaveGotifyInput struct {
	Enabled bool
	URL     string
	Token   string // empty on update keeps existing ciphertext
	Events  []string
}

// SaveWebhookInput updates generic HTTP webhook settings.
type SaveWebhookInput struct {
	Enabled    bool
	URL        string
	BearerToken string // empty on update keeps existing; "-" clears
	Events     []string
}

func normalizeEvents(events []string, fallback string) (string, error) {
	if len(events) == 0 {
		return fallback, nil
	}
	parsed, err := ParseEventsCSV(EventsCSV(events))
	if err != nil {
		return "", err
	}
	if len(parsed) == 0 {
		return fallback, nil
	}
	return EventsCSV(parsed), nil
}

func (s *SettingsStore) ensureRow() error {
	d, err := s.loadRow()
	if err != nil {
		return err
	}
	if d != nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO integration_settings (
			id, slack_enabled, slack_events,
			gotify_enabled, gotify_url, gotify_events,
			webhook_enabled, webhook_url, webhook_events,
			updated_at
		) VALUES (1, 0, ?, 0, '', ?, 0, '', ?, ?)
	`, DefaultSOPEvents, DefaultAllEvents, DefaultAllEvents, now)
	return err
}

// SaveSlack upserts Slack settings. Requires encryption key when setting a new webhook URL.
func (s *SettingsStore) SaveSlack(in SaveSlackInput) error {
	if err := s.ensureRow(); err != nil {
		return err
	}
	d, err := s.loadRow()
	if err != nil {
		return err
	}

	eventsCSV, err := normalizeEvents(in.Events, DefaultSOPEvents)
	if err != nil {
		return err
	}

	urlEnc := d.slackURLEnc
	webhookURL := strings.TrimSpace(in.WebhookURL)
	if webhookURL != "" {
		if !s.KeyConfigured() {
			return ErrKeyMissing
		}
		if err := validateHTTPSURL(webhookURL); err != nil {
			return err
		}
		urlEnc, err = secrets.Seal(s.key, []byte(webhookURL))
		if err != nil {
			return fmt.Errorf("encrypt slack webhook url: %w", err)
		}
	} else if len(urlEnc) == 0 && in.Enabled {
		return ErrSlackWebhookRequired
	}

	enabled := 0
	if in.Enabled {
		enabled = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		UPDATE integration_settings SET
			slack_enabled = ?, slack_webhook_url_enc = ?, slack_events = ?, updated_at = ?
		WHERE id = 1
	`, enabled, urlEnc, eventsCSV, now)
	return err
}

// SaveGotify upserts Gotify settings.
func (s *SettingsStore) SaveGotify(in SaveGotifyInput) error {
	if err := s.ensureRow(); err != nil {
		return err
	}
	d, err := s.loadRow()
	if err != nil {
		return err
	}

	eventsCSV, err := normalizeEvents(in.Events, DefaultAllEvents)
	if err != nil {
		return err
	}

	baseURL := strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if baseURL != "" {
		if err := validateHTTPURL(baseURL); err != nil {
			return err
		}
	}

	tokenEnc := d.gotifyTokenEnc
	token := strings.TrimSpace(in.Token)
	if token != "" {
		if !s.KeyConfigured() {
			return ErrKeyMissing
		}
		tokenEnc, err = secrets.Seal(s.key, []byte(token))
		if err != nil {
			return fmt.Errorf("encrypt gotify token: %w", err)
		}
	}

	if in.Enabled {
		if baseURL == "" {
			return fmt.Errorf("%w: gotify URL is required when enabled", ErrNotConfigured)
		}
		if len(tokenEnc) == 0 {
			return ErrGotifyTokenRequired
		}
	}

	enabled := 0
	if in.Enabled {
		enabled = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		UPDATE integration_settings SET
			gotify_enabled = ?, gotify_url = ?, gotify_token_enc = ?, gotify_events = ?, updated_at = ?
		WHERE id = 1
	`, enabled, baseURL, tokenEnc, eventsCSV, now)
	return err
}

// SaveWebhook upserts generic webhook settings.
// BearerToken "-" clears a previously stored bearer token.
func (s *SettingsStore) SaveWebhook(in SaveWebhookInput) error {
	if err := s.ensureRow(); err != nil {
		return err
	}
	d, err := s.loadRow()
	if err != nil {
		return err
	}

	eventsCSV, err := normalizeEvents(in.Events, DefaultAllEvents)
	if err != nil {
		return err
	}

	hookURL := strings.TrimSpace(in.URL)
	if hookURL != "" {
		if err := validateHTTPURL(hookURL); err != nil {
			return err
		}
	}

	bearerEnc := d.webhookBearerEnc
	switch strings.TrimSpace(in.BearerToken) {
	case "":
		// keep existing
	case "-":
		bearerEnc = nil
	default:
		if !s.KeyConfigured() {
			return ErrKeyMissing
		}
		bearerEnc, err = secrets.Seal(s.key, []byte(strings.TrimSpace(in.BearerToken)))
		if err != nil {
			return fmt.Errorf("encrypt webhook bearer: %w", err)
		}
	}

	if in.Enabled && hookURL == "" {
		return ErrWebhookURLRequired
	}

	enabled := 0
	if in.Enabled {
		enabled = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		UPDATE integration_settings SET
			webhook_enabled = ?, webhook_url = ?, webhook_bearer_enc = ?, webhook_events = ?, updated_at = ?
		WHERE id = 1
	`, enabled, hookURL, bearerEnc, eventsCSV, now)
	return err
}

func validateHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("webhook URL must be a valid https URL")
	}
	return nil
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return fmt.Errorf("URL must be a valid http(s) URL")
	}
	return nil
}

type resolvedSlack struct {
	webhookURL string
	events     string
}

type resolvedGotify struct {
	baseURL string
	token   string
	events  string
}

type resolvedWebhook struct {
	url    string
	bearer string
	events string
}

func (s *SettingsStore) resolveSlack() (*resolvedSlack, error) {
	d, err := s.loadRow()
	if err != nil {
		return nil, err
	}
	if d == nil || d.slackEnabled == 0 {
		return nil, nil
	}
	if len(d.slackURLEnc) == 0 {
		return nil, ErrNotConfigured
	}
	if !s.KeyConfigured() {
		return nil, ErrKeyMissing
	}
	raw, err := secrets.Open(s.key, d.slackURLEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt slack webhook url: %w", err)
	}
	return &resolvedSlack{webhookURL: string(raw), events: d.slackEvents}, nil
}

func (s *SettingsStore) resolveGotify() (*resolvedGotify, error) {
	d, err := s.loadRow()
	if err != nil {
		return nil, err
	}
	if d == nil || d.gotifyEnabled == 0 {
		return nil, nil
	}
	if d.gotifyURL == "" || len(d.gotifyTokenEnc) == 0 {
		return nil, ErrNotConfigured
	}
	if !s.KeyConfigured() {
		return nil, ErrKeyMissing
	}
	raw, err := secrets.Open(s.key, d.gotifyTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt gotify token: %w", err)
	}
	return &resolvedGotify{baseURL: d.gotifyURL, token: string(raw), events: d.gotifyEvents}, nil
}

func (s *SettingsStore) resolveWebhook() (*resolvedWebhook, error) {
	d, err := s.loadRow()
	if err != nil {
		return nil, err
	}
	if d == nil || d.webhookEnabled == 0 {
		return nil, nil
	}
	if d.webhookURL == "" {
		return nil, ErrNotConfigured
	}
	bearer := ""
	if len(d.webhookBearerEnc) > 0 {
		if !s.KeyConfigured() {
			return nil, ErrKeyMissing
		}
		raw, err := secrets.Open(s.key, d.webhookBearerEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypt webhook bearer: %w", err)
		}
		bearer = string(raw)
	}
	return &resolvedWebhook{url: d.webhookURL, bearer: bearer, events: d.webhookEvents}, nil
}
