package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func errorReportingRequest(t *testing.T, env *testenv.Env, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	env.API.ServeHTTP(rec, req)
	return rec
}

// fakeIngest answers like a Sentry/GlitchTip endpoint and returns its DSN.
func fakeIngest(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return "http://public-key@" + strings.TrimPrefix(srv.URL, "http://") + "/42"
}

func adminToken(t *testing.T, env *testenv.Env) string {
	t.Helper()
	env.SeedTestUser(t, "errrep-admin", "errrep-admin@api.local", auth.RoleAdmin)
	token, _ := auth.GenerateAccessToken("errrep-admin", auth.RoleAdmin)
	return token
}

func TestAPI_ErrorReportingPublicConfigIsOffByDefault(t *testing.T) {
	env := testenv.New(t)

	rec := errorReportingRequest(t, env, http.MethodGet, "/api/error-reporting/config", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"enabled":false}` {
		t.Fatalf("body = %s", got)
	}
}

func TestAPI_ErrorReportingSettingsRoundTripAndAudit(t *testing.T) {
	env := testenv.New(t)
	token := adminToken(t, env)
	dsn := "https://secret-looking-key@glitchtip.example.org/7"

	rec := errorReportingRequest(t, env, http.MethodPut, "/api/admin/settings/error-reporting", token, map[string]any{
		"enabled": true, "dsn": dsn, "environment": "lab-a", "browser_enabled": true,
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d %s", rec.Code, rec.Body.String())
	}

	rec = errorReportingRequest(t, env, http.MethodGet, "/api/admin/settings/error-reporting", token, nil)
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["enabled"] != true || got["dsn"] != dsn || got["environment"] != "lab-a" || got["browser_enabled"] != true || got["updated_at"] == "" {
		t.Fatalf("GET = %v", got)
	}

	rec = errorReportingRequest(t, env, http.MethodGet, "/api/error-reporting/config", "", nil)
	var pub map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub["enabled"] != true || pub["dsn"] != dsn || pub["environment"] != "lab-a" || pub["browser"] != true || pub["release"] != "test" {
		t.Fatalf("public config = %v", pub)
	}

	var payload, actor string
	err := env.Store.DB.QueryRow(`SELECT payload, actor_user_id FROM audit_events WHERE event_type = 'error_reporting_settings_updated'`).Scan(&payload, &actor)
	if err != nil {
		t.Fatalf("no audit event: %v", err)
	}
	if actor != "errrep-admin" || !strings.Contains(payload, `"dsn_host":"glitchtip.example.org"`) || !strings.Contains(payload, `"dsn_project":"7"`) {
		t.Fatalf("audit = %s by %s", payload, actor)
	}
	if strings.Contains(payload, "secret-looking-key") {
		t.Fatalf("the audit log must not hold the full DSN: %s", payload)
	}
}

func TestAPI_ErrorReportingRejectsInvalidSettings(t *testing.T) {
	env := testenv.New(t)
	token := adminToken(t, env)
	cases := map[string]map[string]any{
		"dsn_required":        {"enabled": true, "dsn": "", "environment": "production"},
		"dsn_invalid":         {"enabled": true, "dsn": "not a dsn", "environment": "production"},
		"environment_invalid": {"enabled": false, "dsn": "", "environment": "has spaces"},
	}
	for code, body := range cases {
		rec := errorReportingRequest(t, env, http.MethodPut, "/api/admin/settings/error-reporting", token, body)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"error":"`+code+`"`) {
			t.Errorf("%s: got %d %s", code, rec.Code, rec.Body.String())
		}
	}
}

func TestAPI_ErrorReportingIsAdminOnly(t *testing.T) {
	env := testenv.New(t)
	env.SeedTestUser(t, "errrep-auditor", "errrep-auditor@api.local", auth.RoleAuditor)
	token, _ := auth.GenerateAccessToken("errrep-auditor", auth.RoleAuditor)
	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/admin/settings/error-reporting"},
		{http.MethodPut, "/api/admin/settings/error-reporting"},
		{http.MethodPost, "/api/admin/settings/error-reporting/test"},
	} {
		rec := errorReportingRequest(t, env, r.method, r.path, token, map[string]any{})
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s as auditor = %d, want 403", r.method, r.path, rec.Code)
		}
	}
}

func TestAPI_ErrorReportingTestEvent(t *testing.T) {
	env := testenv.New(t)
	token := adminToken(t, env)

	rec := errorReportingRequest(t, env, http.MethodPost, "/api/admin/settings/error-reporting/test", token, nil)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"not_enabled"`) {
		t.Fatalf("while off = %d %s", rec.Code, rec.Body.String())
	}

	put := func(dsn string) {
		rec := errorReportingRequest(t, env, http.MethodPut, "/api/admin/settings/error-reporting", token, map[string]any{
			"enabled": true, "dsn": dsn, "environment": "production",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("PUT = %d %s", rec.Code, rec.Body.String())
		}
	}

	put(fakeIngest(t, http.StatusOK))
	rec = errorReportingRequest(t, env, http.MethodPost, "/api/admin/settings/error-reporting/test", token, nil)
	var ok struct {
		EventID string `json:"event_id"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &ok) != nil || ok.EventID == "" {
		t.Fatalf("delivered = %d %s", rec.Code, rec.Body.String())
	}

	put(fakeIngest(t, http.StatusForbidden))
	rec = errorReportingRequest(t, env, http.MethodPost, "/api/admin/settings/error-reporting/test", token, nil)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), `"delivery_failed"`) {
		t.Fatalf("refused = %d %s", rec.Code, rec.Body.String())
	}

	var delivered, failed int
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'error_reporting_test_sent' AND payload LIKE '%"result":"delivered"%'`).Scan(&delivered)
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'error_reporting_test_sent' AND payload LIKE '%"result":"failed"%'`).Scan(&failed)
	if delivered != 1 || failed != 1 {
		t.Fatalf("audited delivered=%d failed=%d, want 1 and 1", delivered, failed)
	}
}

func TestAPI_ErrorReportingAdminRoutesAreClosedInDemoMode(t *testing.T) {
	env := testenv.New(t, testenv.WithDemoMode())
	token := adminToken(t, env)
	rec := errorReportingRequest(t, env, http.MethodGet, "/api/admin/settings/error-reporting", token, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("demo GET = %d", rec.Code)
	}
	rec = errorReportingRequest(t, env, http.MethodGet, "/api/error-reporting/config", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("demo public config = %d", rec.Code)
	}
}
