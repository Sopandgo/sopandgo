package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

var auditLogRoutes = map[string]bool{
	"GET /api/admin/audit-logs":         true,
	"GET /api/admin/audit-logs/options": true,
}

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

// splitRoute turns a registered pattern such as "DELETE /api/admin/users/{userID}/sessions"
// into a method and a concrete path.
func splitRoute(t *testing.T, pattern string) (method, path string) {
	t.Helper()
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		t.Fatalf("route %q has no method", pattern)
	}
	return method, pathParam.ReplaceAllString(path, "x")
}

// TestAPI_AuditorReadsAuditLogOnly: the audit log needs audit:read, which the
// auditor and admin hold. Every other admin route still needs admin tools.
func TestAPI_AuditorReadsAuditLogOnly(t *testing.T) {
	env := testenv.New(t)

	roles := []string{auth.RoleAdmin, auth.RoleAuditor, auth.RoleViewer, auth.RoleEditor, auth.RoleApprover}
	tokens := map[string]string{}
	for _, role := range roles {
		id := "audit-read-" + role
		env.SeedTestUser(t, id, role+"@audit-read.local", role)
		tokens[role], _ = auth.GenerateAccessToken(id, role)
	}

	var ip int
	do := func(method, path, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, nil)
		ip++
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", ip%250+1)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	for route := range auditLogRoutes {
		method, path := splitRoute(t, route)
		for _, role := range []string{auth.RoleAdmin, auth.RoleAuditor} {
			if rec := do(method, path, tokens[role]); rec.Code != http.StatusOK {
				t.Errorf("%s as %s = %d, want 200: %s", route, role, rec.Code, rec.Body.String())
			}
		}
		for _, role := range []string{auth.RoleViewer, auth.RoleEditor, auth.RoleApprover} {
			if rec := do(method, path, tokens[role]); rec.Code != http.StatusForbidden {
				t.Errorf("%s as %s = %d, want 403", route, role, rec.Code)
			}
		}
	}

	t.Run("filter options list actors for the actor filter", func(t *testing.T) {
		adminID := "audit-read-" + auth.RoleAdmin
		if err := env.AuthService.UpdateUserRole("audit-read-"+auth.RoleViewer, auth.RoleEditor, &adminID); err != nil {
			t.Fatalf("UpdateUserRole: %v", err)
		}
		rec := do(http.MethodGet, "/api/admin/audit-logs/options", tokens[auth.RoleAuditor])
		var options struct {
			Actors []map[string]any `json:"actors"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &options); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(options.Actors) != 1 || options.Actors[0]["id"] != adminID || options.Actors[0]["display_name"] != "Test User" {
			t.Fatalf("actors = %v, want the admin who acted", options.Actors)
		}
		if strings.Contains(rec.Body.String(), "@audit-read.local") {
			t.Errorf("filter options expose emails: %s", rec.Body.String())
		}
	})

	adminRoutes := 0
	for _, route := range env.API.Routes() {
		method, path := splitRoute(t, route)
		if !strings.HasPrefix(path, "/api/admin/") || auditLogRoutes[route] {
			continue
		}
		adminRoutes++
		if rec := do(method, path, tokens[auth.RoleAuditor]); rec.Code != http.StatusForbidden {
			t.Errorf("%s as auditor = %d, want 403", route, rec.Code)
		}
	}
	if adminRoutes < 20 {
		t.Fatalf("found %d admin routes in the route table, expected many more", adminRoutes)
	}
}
