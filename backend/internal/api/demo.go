package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	"golang.org/x/time/rate"
)

// DemoSessionLifetime is how long a demo visitor's refresh token lasts. Refresh
// tokens do not rotate, so a leaked one is useless after this, even between
// nightly resets.
const DemoSessionLifetime = 4 * time.Hour

// demoAccounts maps the role a visitor asks for to the fixed seeded account.
// No other account can be reached without a password.
var demoAccounts = map[string]string{
	auth.RoleViewer:  "researcher@demo.local",
	auth.RoleAuditor: "auditor@demo.local",
}

// ErrDemoReadOnly refuses a request on a read-only public demo.
var ErrDemoReadOnly = errkind.New(errkind.DemoReadOnly, "disabled in the demo")

// demoWriteAllowlist holds the only non-read requests a demo accepts: opening
// a demo session, refreshing its access token, and signing out.
var demoWriteAllowlist = map[string]bool{
	"POST /api/demo/session": true,
	"POST /api/auth/refresh": true,
	"POST /api/auth/logout":  true,
}

// demoAdminReads are the only admin routes a demo answers. They still need
// audit:read through the normal scope check.
var demoAdminReads = map[string]bool{
	"/api/admin/audit-logs":         true,
	"/api/admin/audit-logs/options": true,
}

// demoAllows reports whether a demo instance lets the request through. Reads
// pass, except in the admin area. Writes pass only on the allowlist.
func demoAllows(method, urlPath string) bool {
	// Decide on the cleaned path, so "/api/x/../admin/..." is judged as admin.
	p := path.Clean("/" + urlPath)
	if p == "/api/admin" || strings.HasPrefix(p, "/api/admin/") {
		return method == http.MethodGet && demoAdminReads[p]
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return demoWriteAllowlist[method+" "+p]
}

// withDemoGuard wraps the whole mux on a demo instance. It is modelled on the
// maintenance guard, but runs before routing, so no route can opt out of it.
func withDemoGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !demoAllows(r.Method, r.URL.Path) {
			writeKindJSON(w, ErrDemoReadOnly, "demo_read_only", "Disabled in the demo. Install SOPandGO to try editing.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// registerDemoRoutes adds the routes that exist only on a demo instance.
func (s *Server) registerDemoRoutes() {
	// About ten sessions a minute per visitor address, so one client cannot
	// flood the session table. SvelteKit forwards the visitor's address.
	sessionLimiter := NewIPRateLimiter(rate.Every(6*time.Second), 10)
	s.handle("POST /api/demo/session", sessionLimiter.Middleware(s.handleDemoSession))
}

// handleDemoSession signs a visitor in as the fixed demo account for the
// requested role, without a password, for DemoSessionLifetime.
func (s *Server) handleDemoSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	email, ok := demoAccounts[req.Role]
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_role", "role must be viewer or auditor")
		return
	}

	user, _, err := s.authService.GetUserByEmail(email)
	if err != nil || !user.IsActive {
		log.Printf("ERROR: demo account %s is missing or inactive: %v", email, err)
		http.Error(w, "demo account unavailable", http.StatusServiceUnavailable)
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}
	refreshToken, err := s.authService.CreateSessionWithLifetime(user.ID, DemoSessionLifetime)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	s.writeAudit(audit.EventLogin, audit.EntityUser, user.ID, &user.ID, map[string]string{
		"ip_address": getRealIP(r),
		"user_agent": r.UserAgent(),
		"method":     "demo",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token":       accessToken,
		"refresh_token":      refreshToken,
		"refresh_expires_in": int(DemoSessionLifetime / time.Second),
	})
}
