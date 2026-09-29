package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func testKey() []byte {
	return []byte("01234567890123456789012345678901")
}

func TestParseEventsCSV(t *testing.T) {
	events, err := ParseEventsCSV("sop_published, sop_rc,sop_published,bogus")
	if err == nil {
		t.Fatal("expected error for bogus event")
	}
	events, err = ParseEventsCSV("sop_published, sop_rc,sop_published")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != EventSOPPublished || events[1] != EventSOPRC {
		t.Fatalf("got %#v", events)
	}
}

func TestSettingsStore_SaveAndResolve(t *testing.T) {
	store, err := storage.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()

	ss := NewSettingsStore(store.DB, testKey())
	if err := ss.SaveSlack(SaveSlackInput{
		Enabled:    true,
		WebhookURL: "https://hooks.slack.com/services/T/B/xxx",
		Events:     []string{EventSOPPublished},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ss.SaveGotify(SaveGotifyInput{
		Enabled: true,
		URL:     "http://gotify.local",
		Token:   "tok-1",
		Events:  []string{EventSOPPublished, EventBackupS3Failed},
	}); err != nil {
		t.Fatal(err)
	}
	if err := ss.SaveWebhook(SaveWebhookInput{
		Enabled:     true,
		URL:         "https://example.com/hook",
		BearerToken: "secret",
		Events:      KnownEvents,
	}); err != nil {
		t.Fatal(err)
	}

	pub, err := ss.GetPublic()
	if err != nil {
		t.Fatal(err)
	}
	if !pub.EncryptionKeySet || !pub.Slack.Enabled || !pub.Slack.SecretConfigured {
		t.Fatalf("slack public: %#v", pub.Slack)
	}
	if pub.Gotify.URL != "http://gotify.local" || !pub.Gotify.Configured {
		t.Fatalf("gotify public: %#v", pub.Gotify)
	}
	if pub.Webhook.URL != "https://example.com/hook" || !pub.Webhook.SecretConfigured {
		t.Fatalf("webhook public: %#v", pub.Webhook)
	}

	slack, err := ss.resolveSlack()
	if err != nil || slack == nil || !strings.Contains(slack.webhookURL, "hooks.slack.com") {
		t.Fatalf("resolve slack: %v %#v", err, slack)
	}
}

func TestChannels_HTTP(t *testing.T) {
	var slackBody, gotifyBody, hookBody string
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		body := string(buf[:n])
		switch {
		case strings.Contains(r.URL.Path, "slack"):
			slackBody = body
		case strings.Contains(r.URL.Path, "message"):
			gotifyBody = body
		default:
			hookBody = body
			gotAuth = r.Header.Get("Authorization")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ev := Event{
		Type:       EventSOPPublished,
		Title:      "Published",
		Message:    "Water Protocol v2",
		URL:        "http://lab/sops/1",
		SOPID:      "sop-1",
		Version:    2,
		OccurredAt: time.Now().UTC(),
	}
	ctx := context.Background()

	if err := (&SlackWebhook{WebhookURL: srv.URL + "/slack", Client: srv.Client()}).Notify(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(slackBody, "Published") {
		t.Fatalf("slack body: %s", slackBody)
	}

	if err := (&Gotify{BaseURL: srv.URL, Token: "t", Client: srv.Client()}).Notify(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotifyBody, "Water Protocol") {
		t.Fatalf("gotify body: %s", gotifyBody)
	}

	if err := (&GenericWebhook{URL: srv.URL + "/hook", Bearer: "abc", Client: srv.Client()}).Notify(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer abc" {
		t.Fatalf("auth: %q", gotAuth)
	}
	var env WebhookEnvelope
	if err := json.Unmarshal([]byte(hookBody), &env); err != nil {
		t.Fatal(err)
	}
	if env.Event != EventSOPPublished || env.SOPID != "sop-1" || env.Version != 2 {
		t.Fatalf("envelope: %#v", env)
	}
}

func TestService_Dispatch_RespectsSubscription(t *testing.T) {
	store, err := storage.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ss := NewSettingsStore(store.DB, testKey())
	if err := ss.SaveSlack(SaveSlackInput{
		Enabled:    true,
		WebhookURL: "https://hooks.slack.com/services/T/B/xxx",
		Events:     []string{EventSOPPublished},
	}); err != nil {
		t.Fatal(err)
	}

	// httptest is http; overwrite ciphertext for the local test server.
	encURL, err := secrets.Seal(testKey(), []byte(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Exec(`UPDATE integration_settings SET slack_webhook_url_enc = ? WHERE id = 1`, encURL)
	if err != nil {
		t.Fatal(err)
	}

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(ss, auditLogger)

	svc.Dispatch(context.Background(), Event{Type: EventSOPRC, Title: "RC"})
	if hits != 0 {
		t.Fatalf("expected no send for unsubscribed event, hits=%d", hits)
	}
	svc.Dispatch(context.Background(), Event{Type: EventSOPPublished, Title: "Pub"})
	if hits != 1 {
		t.Fatalf("expected 1 send, hits=%d", hits)
	}
}
