package notify

import (
	"strings"
	"time"
)

// Notification event types (subscription keys). Distinct from audit event_type strings
// where useful; lifecycle hooks map domain actions onto these.
const (
	EventSOPPublished         = "sop_published"
	EventSOPRC                = "sop_rc"
	EventSOPRejected          = "sop_rejected"
	EventBackupS3Failed       = "backup_s3_failed"
	EventIntegrityCheckFailed = "integrity_check_failed"
	EventTest                 = "test"
)

// KnownEvents lists all subscribable event types (excluding test).
var KnownEvents = []string{
	EventSOPPublished,
	EventSOPRC,
	EventSOPRejected,
	EventBackupS3Failed,
	EventIntegrityCheckFailed,
}

var knownEventSet = map[string]struct{}{
	EventSOPPublished:         {},
	EventSOPRC:                {},
	EventSOPRejected:          {},
	EventBackupS3Failed:       {},
	EventIntegrityCheckFailed: {},
	EventTest:                 {},
}

// DefaultSOPEvents is the default subscription for Slack (lifecycle only).
const DefaultSOPEvents = "sop_published,sop_rc,sop_rejected"

// DefaultAllEvents includes lifecycle and ops alerts.
const DefaultAllEvents = "sop_published,sop_rc,sop_rejected,backup_s3_failed,integrity_check_failed"

// Event is a channel-agnostic notification payload.
type Event struct {
	Type       string         `json:"event"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	URL        string         `json:"url,omitempty"`
	SOPID      string         `json:"sop_id,omitempty"`
	VersionID  string         `json:"version_id,omitempty"`
	Version    int            `json:"version,omitempty"`
	ActorID    string         `json:"actor_id,omitempty"`
	ActorName  string         `json:"actor,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// IsKnownEvent reports whether t is a valid notification event type.
func IsKnownEvent(t string) bool {
	_, ok := knownEventSet[t]
	return ok
}

// ParseEventsCSV splits a comma-separated event list, validates known types, and returns unique ordered values.
func ParseEventsCSV(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		e := strings.TrimSpace(p)
		if e == "" {
			continue
		}
		if e == EventTest {
			continue // test is never a stored subscription
		}
		if !IsKnownEvent(e) {
			return nil, ErrInvalidEvent
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out, nil
}

// EventsCSV joins event types for storage.
func EventsCSV(events []string) string {
	return strings.Join(events, ",")
}

// Subscribes reports whether csv includes eventType (or EventTest always matches for tests via separate path).
func Subscribes(csv, eventType string) bool {
	if eventType == EventTest {
		return true
	}
	for _, e := range strings.Split(csv, ",") {
		if strings.TrimSpace(e) == eventType {
			return true
		}
	}
	return false
}
