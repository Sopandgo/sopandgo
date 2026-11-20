package api

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// redacted replaces visitor data in a demo's audit responses.
const redacted = "redacted"

// redactedKeys are payload fields that hold a visitor's address, browser or
// email, whatever their value looks like.
var redactedKeys = map[string]bool{
	"ip_address":      true,
	"user_agent":      true,
	"attempted_email": true,
	"email":           true,
	"recipient":       true,
}

// emailPattern finds an email address anywhere in a string, so an email under
// any other key, or as an entity id, is redacted too.
var emailPattern = regexp.MustCompile(`[^\s@"'<>(),;:]+@[^\s@"'<>(),;:]+`)

// redactAuditEvents rewrites events for a public demo response. Stored rows
// are unchanged; hash_valid was computed on them before this runs.
func redactAuditEvents(events []audit.AuditEventWithVerification) []audit.AuditEventWithVerification {
	out := make([]audit.AuditEventWithVerification, len(events))
	for i, ev := range events {
		ev.EntityID = emailPattern.ReplaceAllString(ev.EntityID, redacted)
		ev.Payload = redactPayload(ev.Payload)
		out[i] = ev
	}
	return out
}

func redactPayload(payload string) string {
	var v any
	if err := json.Unmarshal([]byte(payload), &v); err != nil {
		return emailPattern.ReplaceAllString(payload, redacted)
	}
	b, err := json.Marshal(redactValue("", v))
	if err != nil {
		return redacted
	}
	return string(b)
}

func redactValue(key string, v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, inner := range val {
			val[k] = redactValue(k, inner)
		}
		return val
	case []any:
		for i, inner := range val {
			val[i] = redactValue(key, inner)
		}
		return val
	case string:
		if redactedKeys[key] || strings.HasSuffix(key, "_email") {
			return redacted
		}
		return emailPattern.ReplaceAllString(val, redacted)
	default:
		return v
	}
}
