package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// Password sign-in is limited per account as well as per address: failed
// attempts at one email from many addresses stop after a burst of ten.
func TestAPI_LoginLimitedPerAccount(t *testing.T) {
	env := testenv.New(t)
	adminID := "admin-actor"
	env.SeedTestUser(t, adminID, "admin@api.local", auth.RoleAdmin)

	const password = "SuperSecret123!"
	newUser := func(email string) {
		t.Helper()
		id, err := env.AuthService.RegisterUser("Test User", email, auth.RoleEditor, &adminID)
		if err != nil {
			t.Fatal(err)
		}
		if err := env.AuthService.UpdateUserPassword(id, password, &adminID); err != nil {
			t.Fatal(err)
		}
	}
	newUser("target@limit.local")
	newUser("bystander@limit.local")
	newUser("frequent@limit.local")

	// Every attempt comes from a new address, as in a distributed attack, so
	// the per-address limiter never applies.
	addr := 0
	login := func(email, pw string) int {
		addr++
		b, _ := json.Marshal(map[string]string{"email": email, "password": pw})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = fmt.Sprintf("198.18.%d.%d:1234", addr/250, addr%250+1)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("ten failures, then even the right password waits", func(t *testing.T) {
		for i := range 10 {
			if code := login("target@limit.local", "wrong"); code != http.StatusUnauthorized {
				t.Fatalf("attempt %d: status %d, want 401", i+1, code)
			}
		}
		if code := login("target@limit.local", password); code != http.StatusTooManyRequests {
			t.Fatalf("status %d, want 429", code)
		}
		if code := login("  TARGET@limit.local ", password); code != http.StatusTooManyRequests {
			t.Fatalf("case and spaces: status %d, want 429", code)
		}

		var raw string
		if err := env.Store.DB.QueryRow(
			`SELECT payload FROM audit_events WHERE event_type = ? ORDER BY id DESC LIMIT 1`,
			string(audit.EventLoginFailed),
		).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var p map[string]string
		_ = json.Unmarshal([]byte(raw), &p)
		if p["failure_reason"] != "too_many_attempts" {
			t.Errorf("failure_reason = %q, want too_many_attempts", p["failure_reason"])
		}
	})

	t.Run("other accounts are unaffected", func(t *testing.T) {
		if code := login("bystander@limit.local", password); code != http.StatusOK {
			t.Fatalf("status %d, want 200", code)
		}
	})

	t.Run("successful sign-ins do not count", func(t *testing.T) {
		for i := range 15 {
			if code := login("frequent@limit.local", password); code != http.StatusOK {
				t.Fatalf("sign-in %d: status %d, want 200", i+1, code)
			}
		}
	})

	t.Run("unknown emails are limited the same way", func(t *testing.T) {
		for range 10 {
			login("nobody@limit.local", "wrong")
		}
		if code := login("nobody@limit.local", "wrong"); code != http.StatusTooManyRequests {
			t.Fatalf("status %d, want 429", code)
		}
	})
}
