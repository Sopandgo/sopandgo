package notify

import (
	"context"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// Channel sends a single notification to one destination.
type Channel interface {
	Name() string
	Notify(ctx context.Context, ev Event) error
}

// Auditor writes audit events (satisfied by *audit.Logger).
type Auditor interface {
	Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
}

// Channel names used in audit payloads and public settings.
const (
	ChannelSlack   = "slack"
	ChannelGotify  = "gotify"
	ChannelWebhook = "webhook"
)
