package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// Sign-in audit events record the visitor's address, resolved like the rate
// limiter does: the forwarded hop from the in-container proxy (Caddy or
// SvelteKit on loopback), otherwise the peer without its port.
func TestAPI_LoginAuditRecordsVisitorAddress(t *testing.T) {
	env := testenv.New(t)
	adminID := "admin-actor"
	env.SeedTestUser(t, adminID, "admin@api.local", auth.RoleAdmin)

	email := "user@audit-ip.local"
	password := "SuperSecret123!"
	userID, err := env.AuthService.RegisterUser("Test User", email, auth.RoleEditor, &adminID)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.AuthService.UpdateUserPassword(userID, password, &adminID); err != nil {
		t.Fatal(err)
	}

	login := func(remoteAddr, forwardedFor, pw string) int {
		b, _ := json.Marshal(map[string]string{"email": email, "password": pw})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TestBrowser/1.0")
		req.RemoteAddr = remoteAddr
		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec.Code
	}

	lastPayload := func(eventType string) map[string]string {
		t.Helper()
		var raw string
		err := env.Store.DB.QueryRow(
			`SELECT payload FROM audit_events WHERE event_type = ? ORDER BY id DESC LIMIT 1`, eventType,
		).Scan(&raw)
		if err != nil {
			t.Fatalf("no %s event: %v", eventType, err)
		}
		var p map[string]string
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatalf("payload %q: %v", raw, err)
		}
		return p
	}

	t.Run("failed sign-in through the proxy", func(t *testing.T) {
		if code := login("127.0.0.1:41000", "203.0.113.50", "wrong"); code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", code)
		}
		p := lastPayload(string(audit.EventLoginFailed))
		if p["ip_address"] != "203.0.113.50" {
			t.Errorf("ip_address = %q, want the forwarded visitor", p["ip_address"])
		}
		if p["user_agent"] != "TestBrowser/1.0" {
			t.Errorf("user_agent = %q", p["user_agent"])
		}
	})

	t.Run("successful sign-in through the proxy", func(t *testing.T) {
		if code := login("127.0.0.1:41001", "198.51.100.60", password); code != http.StatusOK {
			t.Fatalf("status %d, want 200", code)
		}
		if got := lastPayload(string(audit.EventLogin))["ip_address"]; got != "198.51.100.60" {
			t.Errorf("ip_address = %q, want the forwarded visitor", got)
		}
	})

	t.Run("direct peer cannot choose its address", func(t *testing.T) {
		if code := login("192.0.2.70:5555", "1.2.3.4", "wrong"); code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", code)
		}
		if got := lastPayload(string(audit.EventLoginFailed))["ip_address"]; got != "192.0.2.70" {
			t.Errorf("ip_address = %q, want the peer without its port", got)
		}
	})
}
