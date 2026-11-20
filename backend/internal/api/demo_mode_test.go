package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// demoWriteAllowlist is every non-GET route the demo accepts.
var demoWriteAllowlist = map[string]bool{
	"POST /api/demo/session": true,
	"POST /api/auth/refresh": true,
	"POST /api/auth/logout":  true,
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// demoClient sends requests the way SvelteKit does in the container: from
// loopback, with the visitor's address as the last X-Forwarded-For hop.
type demoClient struct {
	env *testenv.Env
	ip  atomic.Int64
}

func (c *demoClient) do(method, path, token string, body any) *httptest.ResponseRecorder {
	n := c.ip.Add(1)
	return c.doFrom(fmt.Sprintf("203.0.113.%d", n%250+1), method, path, token, body)
}

func (c *demoClient) doFrom(clientIP, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "127.0.0.1:41000"
	req.Header.Set("X-Forwarded-For", clientIP)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	c.env.API.ServeHTTP(rec, req)
	return rec
}

func errorCode(rec *httptest.ResponseRecorder) string {
	var body struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return body.Error
}

// demoEnv is a demo-mode server with the two fixed demo accounts and a token
// for every role.
func demoEnv(t *testing.T) (*testenv.Env, *demoClient, map[string]string) {
	t.Helper()
	env := testenv.New(t, testenv.WithDemoMode())
	env.SeedTestUser(t, "demo-researcher", "researcher@demo.local", auth.RoleViewer)
	env.SeedTestUser(t, "demo-auditor", "auditor@demo.local", auth.RoleAuditor)

	tokens := map[string]string{}
	for _, role := range []string{auth.RoleAdmin, auth.RoleEditor, auth.RoleApprover, auth.RoleAuditor, auth.RoleViewer} {
		id := "demo-role-" + role
		env.SeedTestUser(t, id, role+"@demo-role.local", role)
		tokens[role], _ = auth.GenerateAccessToken(id, role)
	}
	return env, &demoClient{env: env}, tokens
}

// TestDemoMode_RefusesEveryWrite: with demo mode on, the API itself refuses
// every write outside the allowlist, for every role including admin, and the
// whole admin area except the audit log reads.
func TestDemoMode_RefusesEveryWrite(t *testing.T) {
	env, c, tokens := demoEnv(t)

	checked := 0
	for _, route := range env.API.Routes() {
		method, path := splitRoute(t, route)
		admin := strings.HasPrefix(path, "/api/admin/")
		if demoWriteAllowlist[route] || (isReadMethod(method) && !admin) || auditLogRoutes[route] {
			continue
		}
		checked++
		for role, token := range tokens {
			rec := c.do(method, path, token, map[string]string{})
			if rec.Code != http.StatusForbidden || errorCode(rec) != "demo_read_only" {
				t.Errorf("%s as %s = %d %q, want 403 demo_read_only", route, role, rec.Code, errorCode(rec))
			}
		}
	}
	if checked < 50 {
		t.Fatalf("checked %d routes, expected the whole write surface", checked)
	}
	if env.MockMailSender.SentCount != 0 {
		t.Errorf("the write sweep sent %d emails", env.MockMailSender.SentCount)
	}

	t.Run("sign-in without the demo endpoint is refused", func(t *testing.T) {
		for _, path := range []string{
			"/api/auth/login",
			"/api/auth/login/two-factor",
			"/api/auth/login/two-factor/email",
		} {
			rec := c.do(http.MethodPost, path, "", map[string]string{"email": "admin", "password": "admin"})
			if rec.Code != http.StatusForbidden || errorCode(rec) != "demo_read_only" {
				t.Errorf("POST %s = %d, want 403 demo_read_only", path, rec.Code)
			}
		}
		rec := c.do(http.MethodPatch, "/api/auth/reset-password", "", map[string]string{})
		if rec.Code != http.StatusForbidden {
			t.Errorf("PATCH reset-password = %d, want 403", rec.Code)
		}
	})

	t.Run("cleaned or escaped paths cannot reach the admin area", func(t *testing.T) {
		for _, path := range []string{
			"/api/sops/../admin/backups/export",
			"/api//admin/backups/export",
			"/api/%61dmin/backups/export",
		} {
			rec := c.do(http.MethodGet, path, tokens[auth.RoleAdmin], nil)
			if rec.Code == http.StatusOK {
				t.Errorf("GET %s = 200, want refused or redirected", path)
			}
		}
	})

	t.Run("audit log reads still need audit:read", func(t *testing.T) {
		for route := range auditLogRoutes {
			method, path := splitRoute(t, route)
			if rec := c.do(method, path, tokens[auth.RoleAuditor], nil); rec.Code != http.StatusOK {
				t.Errorf("%s as auditor = %d, want 200", route, rec.Code)
			}
			if rec := c.do(method, path, tokens[auth.RoleViewer], nil); rec.Code != http.StatusForbidden {
				t.Errorf("%s as viewer = %d, want 403", route, rec.Code)
			}
		}
	})

	t.Run("reads outside the admin area work", func(t *testing.T) {
		for _, path := range []string{"/api/sops", "/api/tags", "/api/auth/me", "/api/activity/publishes"} {
			if rec := c.do(http.MethodGet, path, tokens[auth.RoleViewer], nil); rec.Code != http.StatusOK {
				t.Errorf("GET %s = %d, want 200", path, rec.Code)
			}
		}
	})
}

type demoSession struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

func openDemoSession(t *testing.T, c *demoClient, role string) demoSession {
	t.Helper()
	rec := c.do(http.MethodPost, "/api/demo/session", "", map[string]string{"role": role})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/demo/session %s = %d: %s", role, rec.Code, rec.Body.String())
	}
	var s demoSession
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode demo session: %v", err)
	}
	return s
}

// TestDemoMode_SessionEndpoint: visitors get a session for one of the two
// fixed demo accounts without a password, for four hours.
func TestDemoMode_SessionEndpoint(t *testing.T) {
	env, c, tokens := demoEnv(t)

	for role, email := range map[string]string{"viewer": "researcher@demo.local", "auditor": "auditor@demo.local"} {
		t.Run("opens a working session as "+role, func(t *testing.T) {
			s := openDemoSession(t, c, role)
			me := c.do(http.MethodGet, "/api/auth/me", s.AccessToken, nil)
			if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), email) {
				t.Fatalf("/api/auth/me = %d %s, want %s", me.Code, me.Body.String(), email)
			}
			if s.RefreshExpiresIn != 4*60*60 {
				t.Errorf("refresh_expires_in = %d, want 4 hours", s.RefreshExpiresIn)
			}
		})
	}

	t.Run("refuses any other role", func(t *testing.T) {
		for _, role := range []string{"admin", "editor", "approver", "", "Viewer"} {
			rec := c.do(http.MethodPost, "/api/demo/session", "", map[string]string{"role": role})
			if rec.Code != http.StatusBadRequest {
				t.Errorf("role %q = %d, want 400", role, rec.Code)
			}
		}
	})

	t.Run("the refresh token lasts the demo lifetime", func(t *testing.T) {
		before := time.Now()
		s := openDemoSession(t, c, "viewer")
		sessions, err := env.AuthService.ListActiveSessions()
		if err != nil {
			t.Fatalf("ListActiveSessions: %v", err)
		}
		found := false
		for _, sess := range sessions {
			if sess.UserID != "demo-researcher" || sess.CreatedAt.Before(before.Add(-time.Second)) {
				continue
			}
			found = true
			lifetime := sess.ExpiresAt.Sub(sess.CreatedAt)
			if lifetime != 4*time.Hour {
				t.Errorf("demo session lifetime = %s, want 4h", lifetime)
			}
		}
		if !found {
			t.Fatalf("no session stored for the demo session %s", s.RefreshToken[:4])
		}
	})

	t.Run("refresh and logout work", func(t *testing.T) {
		s := openDemoSession(t, c, "auditor")
		body := map[string]string{"refresh_token": s.RefreshToken}
		if rec := c.do(http.MethodPost, "/api/auth/refresh", "", body); rec.Code != http.StatusOK {
			t.Fatalf("refresh = %d: %s", rec.Code, rec.Body.String())
		}
		if rec := c.do(http.MethodPost, "/api/auth/logout", "", body); rec.Code != http.StatusNoContent {
			t.Fatalf("logout = %d: %s", rec.Code, rec.Body.String())
		}
		if rec := c.do(http.MethodPost, "/api/auth/refresh", "", body); rec.Code != http.StatusUnauthorized {
			t.Fatalf("refresh after logout = %d, want 401", rec.Code)
		}
	})

	t.Run("writes the login audit event for the account", func(t *testing.T) {
		openDemoSession(t, c, "viewer")
		rec := c.do(http.MethodGet, "/api/admin/audit-logs?type=login&actor_user_id=demo-researcher", tokens[auth.RoleAuditor], nil)
		var res struct {
			Total int `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if rec.Code != http.StatusOK || res.Total == 0 {
			t.Fatalf("login events for the researcher = %d (status %d), want at least one", res.Total, rec.Code)
		}
	})

	t.Run("is rate-limited per forwarded client address", func(t *testing.T) {
		limited := false
		for i := 0; i < 15; i++ {
			rec := c.doFrom("198.18.0.7", http.MethodPost, "/api/demo/session", "", map[string]string{"role": "viewer"})
			if rec.Code == http.StatusTooManyRequests {
				limited = true
				break
			}
		}
		if !limited {
			t.Fatal("15 demo sessions from one address were never rate-limited")
		}
		rec := c.doFrom("198.18.0.8", http.MethodPost, "/api/demo/session", "", map[string]string{"role": "viewer"})
		if rec.Code != http.StatusOK {
			t.Fatalf("another address = %d, want 200", rec.Code)
		}
	})
}

// TestDemoMode_OffByDefault: without demo mode there is no password-free
// session endpoint and nothing is redacted.
func TestDemoMode_OffByDefault(t *testing.T) {
	env := testenv.New(t)
	env.SeedTestUser(t, "demo-researcher", "researcher@demo.local", auth.RoleViewer)
	c := &demoClient{env: env}
	rec := c.do(http.MethodPost, "/api/demo/session", "", map[string]string{"role": "viewer"})
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/demo/session without demo mode = %d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "access_token") {
		t.Fatalf("demo session issued without demo mode: %s", rec.Body.String())
	}
}

// visitorTraces are personal data a public demo must not show back.
var visitorTraces = []string{"visitor@example.org", "203.0.113.99", "VisitorBrowser/1.0", "new@lab.example", "inbox@lab.example", "nested@lab.example"}

func logVisitorTraces(t *testing.T, env *testenv.Env) {
	t.Helper()
	events := []struct {
		eventType, entityType, entityID string
		payload                         any
	}{
		{"login_failed", "user", "visitor@example.org", map[string]string{
			"attempted_email": "visitor@example.org", "ip_address": "203.0.113.99",
			"user_agent": "VisitorBrowser/1.0", "failure_reason": "user_not_found",
		}},
		{"user_created", "user", "u-1", map[string]string{"email": "new@lab.example", "role": "viewer"}},
		{"email_sent", "user", "u-1", map[string]string{"recipient": "inbox@lab.example", "status": "sent"}},
		{"user_role_updated", "user", "u-1", map[string]any{"target": map[string]string{"contact": "nested@lab.example"}}},
	}
	for _, ev := range events {
		if err := env.AuditLogger.Log(nil, ev.eventType, ev.entityType, ev.entityID, nil, ev.payload); err != nil {
			t.Fatalf("audit log %s: %v", ev.eventType, err)
		}
	}
}

// TestDemoMode_RedactsAuditLog: on a demo, audit responses replace visitors'
// addresses, browsers and emails with "redacted", for every role.
func TestDemoMode_RedactsAuditLog(t *testing.T) {
	env, c, tokens := demoEnv(t)
	logVisitorTraces(t, env)
	openDemoSession(t, c, "viewer")

	for _, role := range []string{auth.RoleAdmin, auth.RoleAuditor} {
		rec := c.do(http.MethodGet, "/api/admin/audit-logs?limit=100", tokens[role], nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("audit log as %s = %d", role, rec.Code)
		}
		body := rec.Body.String()
		for _, trace := range visitorTraces {
			if strings.Contains(body, trace) {
				t.Errorf("audit log as %s shows %q", role, trace)
			}
		}
		if strings.Contains(body, "203.0.113.") {
			t.Errorf("audit log as %s shows a visitor address: %s", role, body)
		}
		if !strings.Contains(body, `\"ip_address\":\"redacted\"`) || !strings.Contains(body, `"entity_id":"redacted"`) {
			t.Errorf("audit log as %s does not say redacted: %s", role, body)
		}
		var res struct {
			Events []struct {
				HashValid bool `json:"hash_valid"`
			} `json:"events"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		for _, ev := range res.Events {
			if !ev.HashValid {
				t.Errorf("redaction broke hash verification as %s", role)
			}
		}
	}
}

// TestDemoMode_NoRedactionWhenOff: real auditors and admins see addresses
// and browsers, which they need to investigate incidents.
func TestDemoMode_NoRedactionWhenOff(t *testing.T) {
	env := testenv.New(t)
	env.SeedTestUser(t, "real-auditor", "auditor@real.local", auth.RoleAuditor)
	token, _ := auth.GenerateAccessToken("real-auditor", auth.RoleAuditor)
	logVisitorTraces(t, env)

	rec := (&demoClient{env: env}).do(http.MethodGet, "/api/admin/audit-logs?limit=100", token, nil)
	for _, trace := range visitorTraces {
		if !strings.Contains(rec.Body.String(), trace) {
			t.Errorf("audit log without demo mode hides %q", trace)
		}
	}
}

// TestDemoMode_SendsNothing: on a demo, mail and notifications are no-ops in
// Go, whatever the stored settings say.
func TestDemoMode_SendsNothing(t *testing.T) {
	cases := []struct {
		name     string
		opts     []testenv.Option
		wantSent bool
	}{
		{"demo mode", []testenv.Option{testenv.WithDemoMode()}, false},
		{"normal install", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := testenv.New(t, tc.opts...)

			var hits atomic.Int64
			hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
			}))
			t.Cleanup(hook.Close)
			if err := env.IntegrationSettings.SaveWebhook(notify.SaveWebhookInput{Enabled: true, URL: hook.URL}); err != nil {
				t.Fatalf("SaveWebhook: %v", err)
			}

			env.Notify.Dispatch(env.Ctx, notify.Event{Type: notify.EventSOPPublished, Title: "Published", Message: "v1"})
			_ = env.MailService.SendTwoFactorCodeEmail("someone@lab.example", "u-1", "Someone", "123456", "en")

			sent := hits.Load() > 0 || env.MockMailSender.SentCount > 0
			if sent != tc.wantSent {
				t.Fatalf("webhook hits %d, mails %d; want sent=%v", hits.Load(), env.MockMailSender.SentCount, tc.wantSent)
			}
			if tc.wantSent && (hits.Load() == 0 || env.MockMailSender.SentCount == 0) {
				t.Fatalf("control did not reach both channels: hits %d, mails %d", hits.Load(), env.MockMailSender.SentCount)
			}
		})
	}
}
