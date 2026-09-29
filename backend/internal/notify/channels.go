package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 10 * time.Second

func httpClient() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

// SlackWebhook posts Incoming Webhook JSON {"text": "..."}.
type SlackWebhook struct {
	WebhookURL string
	Client     *http.Client
}

func (s *SlackWebhook) Name() string { return ChannelSlack }

func (s *SlackWebhook) Notify(ctx context.Context, ev Event) error {
	text := formatPlainText(ev)
	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = httpClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("slack webhook status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// Gotify posts to {baseURL}/message?token=...
type Gotify struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (g *Gotify) Name() string { return ChannelGotify }

func (g *Gotify) Notify(ctx context.Context, ev Event) error {
	priority := 5
	switch ev.Type {
	case EventBackupS3Failed, EventIntegrityCheckFailed:
		priority = 8
	case EventTest:
		priority = 2
	}
	payload := map[string]any{
		"title":    ev.Title,
		"message":  formatPlainText(ev),
		"priority": priority,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(g.BaseURL, "/") + "/message?token=" + g.Token
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = httpClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("gotify status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// GenericWebhook POSTs a stable JSON envelope to an arbitrary URL.
type GenericWebhook struct {
	URL    string
	Bearer string
	Client *http.Client
}

func (w *GenericWebhook) Name() string { return ChannelWebhook }

// WebhookEnvelope is the stable JSON body for generic HTTP webhooks.
type WebhookEnvelope struct {
	Event      string         `json:"event"`
	OccurredAt string         `json:"occurred_at"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	URL        string         `json:"url,omitempty"`
	SOPID      string         `json:"sop_id,omitempty"`
	VersionID  string         `json:"version_id,omitempty"`
	Version    int            `json:"version,omitempty"`
	ActorID    string         `json:"actor_id,omitempty"`
	Actor      string         `json:"actor,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

func (w *GenericWebhook) Notify(ctx context.Context, ev Event) error {
	env := WebhookEnvelope{
		Event:      ev.Type,
		OccurredAt: ev.OccurredAt.UTC().Format(time.RFC3339),
		Title:      ev.Title,
		Message:    ev.Message,
		URL:        ev.URL,
		SOPID:      ev.SOPID,
		VersionID:  ev.VersionID,
		Version:    ev.Version,
		ActorID:    ev.ActorID,
		Actor:      ev.ActorName,
		Extra:      ev.Extra,
	}
	if env.Message == "" {
		env.Message = formatPlainText(ev)
	}
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if w.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+w.Bearer)
	}
	client := w.Client
	if client == nil {
		client = httpClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func formatPlainText(ev Event) string {
	var b strings.Builder
	if ev.Title != "" {
		b.WriteString(ev.Title)
	}
	if ev.Message != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(ev.Message)
	}
	if ev.URL != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(ev.URL)
	}
	if b.Len() == 0 {
		return ev.Type
	}
	return b.String()
}
