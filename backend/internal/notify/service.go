package notify

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
)

// Service fans out notifications to enabled, subscribed channels.
type Service struct {
	store   *SettingsStore
	auditor Auditor
}

// NewService creates a notification fan-out service.
func NewService(store *SettingsStore, auditor Auditor) *Service {
	return &Service{store: store, auditor: auditor}
}

// Dispatch sends ev to every enabled channel that subscribes to ev.Type.
// Errors are audited and logged; they never propagate to the caller.
func (s *Service) Dispatch(ctx context.Context, ev Event) {
	if s == nil || s.store == nil {
		return
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	if !IsKnownEvent(ev.Type) {
		log.Printf("notify: unknown event type %q", ev.Type)
		return
	}

	type job struct {
		name string
		ch   Channel
	}
	var jobs []job

	if slack, err := s.store.resolveSlack(); err != nil {
		s.auditFail(ChannelSlack, ev, err)
	} else if slack != nil && Subscribes(slack.events, ev.Type) {
		jobs = append(jobs, job{ChannelSlack, &SlackWebhook{WebhookURL: slack.webhookURL}})
	}

	if gotify, err := s.store.resolveGotify(); err != nil {
		s.auditFail(ChannelGotify, ev, err)
	} else if gotify != nil && Subscribes(gotify.events, ev.Type) {
		jobs = append(jobs, job{ChannelGotify, &Gotify{BaseURL: gotify.baseURL, Token: gotify.token}})
	}

	if hook, err := s.store.resolveWebhook(); err != nil {
		s.auditFail(ChannelWebhook, ev, err)
	} else if hook != nil && Subscribes(hook.events, ev.Type) {
		jobs = append(jobs, job{ChannelWebhook, &GenericWebhook{URL: hook.url, Bearer: hook.bearer}})
	}

	if len(jobs) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
			defer cancel()
			if err := j.ch.Notify(cctx, ev); err != nil {
				log.Printf("notify %s failed for %s: %v", j.name, ev.Type, err)
				s.auditFail(j.name, ev, err)
				return
			}
			s.auditOK(j.name, ev)
		}(j)
	}
	wg.Wait()
}

// SendTest sends a test notification to a single channel using stored credentials
// (works even when the channel is currently disabled).
func (s *Service) SendTest(ctx context.Context, channel, locale string) error {
	if s == nil || s.store == nil {
		return ErrNotConfigured
	}
	ev := Event{
		Type:       EventTest,
		Title:      i18n.T(locale, "notify.test.title", nil),
		Message:    i18n.T(locale, "notify.test.message", map[string]string{"channel": channel}),
		OccurredAt: time.Now().UTC(),
	}

	ch, err := s.channelForTest(channel)
	if err != nil {
		return err
	}

	cctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()
	if err := ch.Notify(cctx, ev); err != nil {
		s.auditFail(channel, ev, err)
		return err
	}
	s.auditOK(channel, ev)
	return nil
}

func (s *Service) channelForTest(channel string) (Channel, error) {
	switch channel {
	case ChannelSlack:
		slack, err := s.store.loadSlackCredentials()
		if err != nil {
			return nil, err
		}
		return &SlackWebhook{WebhookURL: slack}, nil
	case ChannelGotify:
		base, token, err := s.store.loadGotifyCredentials()
		if err != nil {
			return nil, err
		}
		return &Gotify{BaseURL: base, Token: token}, nil
	case ChannelWebhook:
		url, bearer, err := s.store.loadWebhookCredentials()
		if err != nil {
			return nil, err
		}
		return &GenericWebhook{URL: url, Bearer: bearer}, nil
	default:
		return nil, fmt.Errorf("%w: unknown channel %q", ErrNotConfigured, channel)
	}
}

func (s *SettingsStore) loadSlackCredentials() (string, error) {
	d, err := s.loadRow()
	if err != nil {
		return "", err
	}
	if d == nil || len(d.slackURLEnc) == 0 {
		return "", ErrNotConfigured
	}
	if !s.KeyConfigured() {
		return "", ErrKeyMissing
	}
	raw, err := secrets.Open(s.key, d.slackURLEnc)
	if err != nil {
		return "", fmt.Errorf("decrypt slack webhook url: %w", err)
	}
	return string(raw), nil
}

func (s *SettingsStore) loadGotifyCredentials() (baseURL, token string, err error) {
	d, err := s.loadRow()
	if err != nil {
		return "", "", err
	}
	if d == nil || d.gotifyURL == "" || len(d.gotifyTokenEnc) == 0 {
		return "", "", ErrNotConfigured
	}
	if !s.KeyConfigured() {
		return "", "", ErrKeyMissing
	}
	raw, err := secrets.Open(s.key, d.gotifyTokenEnc)
	if err != nil {
		return "", "", fmt.Errorf("decrypt gotify token: %w", err)
	}
	return d.gotifyURL, string(raw), nil
}

func (s *SettingsStore) loadWebhookCredentials() (hookURL, bearer string, err error) {
	d, err := s.loadRow()
	if err != nil {
		return "", "", err
	}
	if d == nil || d.webhookURL == "" {
		return "", "", ErrNotConfigured
	}
	if len(d.webhookBearerEnc) > 0 {
		if !s.KeyConfigured() {
			return "", "", ErrKeyMissing
		}
		raw, err := secrets.Open(s.key, d.webhookBearerEnc)
		if err != nil {
			return "", "", fmt.Errorf("decrypt webhook bearer: %w", err)
		}
		bearer = string(raw)
	}
	return d.webhookURL, bearer, nil
}

func (s *Service) auditOK(channel string, ev Event) {
	if s.auditor == nil {
		return
	}
	_ = s.auditor.Log(nil, audit.EventNotificationSent, audit.EntityNotification, channel, nil, map[string]any{
		"channel": channel,
		"event":   ev.Type,
		"sop_id":  ev.SOPID,
	})
}

func (s *Service) auditFail(channel string, ev Event, err error) {
	if s.auditor == nil {
		return
	}
	_ = s.auditor.Log(nil, audit.EventNotificationFailed, audit.EntityNotification, channel, nil, map[string]any{
		"channel": channel,
		"event":   ev.Type,
		"sop_id":  ev.SOPID,
		"error":   err.Error(),
	})
}
