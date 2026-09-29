package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"golang.org/x/time/rate"
)

// Define a custom type for context keys to avoid collisions
type contextKey string

const userIDKey contextKey = "userID"
const userRoleKey contextKey = "userRole"

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.VerifyToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Use the claims by adding them to the request context
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, userRoleKey, claims.UserRole)

		// Pass the new context to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// withPasswordChangeGuard loads the account on every protected request.
// Inactive users are rejected, and authorization uses the stored role.
// Accounts flagged to change their password may only call GET /api/auth/me
// and PATCH /api/auth/me/update-password.
func (s *Server) withPasswordChangeGuard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
		user, err := s.authService.GetUserByID(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}
		if !user.IsActive {
			http.Error(w, "Account is inactive", http.StatusUnauthorized)
			return
		}

		// Authorize with the stored role. The access token's role copy can be
		// stale for up to 5 minutes after a demotion.
		ctx := context.WithValue(r.Context(), userRoleKey, user.Role)
		r = r.WithContext(ctx)

		if user.MustChangePassword {
			path := r.URL.Path
			allowed := (r.Method == http.MethodGet && path == "/api/auth/me") ||
				(r.Method == http.MethodPatch && path == "/api/auth/me/update-password")
			if !allowed {
				http.Error(w, "password change required", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	}
}

func (s *Server) withMaintenanceGuard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow backup endpoints to progress while lock is active.
		if strings.HasPrefix(r.URL.Path, "/api/admin/backups/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		locked, msg := s.backupSvc.IsLocked()
		if locked {
			if msg == "" {
				msg = "maintenance lock is active"
			}
			w.Header().Set("Retry-After", "15")
			http.Error(w, msg, http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// GetUserID retrieves the authenticated user's ID from the request context.
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// GetUserRole retrieves the authenticated user's role from the request context.
func GetUserRole(ctx context.Context) string {
	role, _ := ctx.Value(userRoleKey).(string)
	return role
}

func (s *Server) requireScope(scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userRole := GetUserRole(r.Context()) //

		if !auth.HasPermission(userRole, scope) {
			http.Error(w, "forbidden: missing scope "+scope, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips map[string]*client
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter starts the cleanup routine automatically
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*client),
		r:   r,
		b:   b,
	}

	// Start a background goroutine to clean up every minute
	go i.cleanupLoop()

	return i
}

func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry, exists := i.ips[ip]
	if !exists {
		entry = &client{
			limiter: rate.NewLimiter(i.r, i.b),
		}
		i.ips[ip] = entry
	}

	// Update the last seen time whenever the user makes a request
	entry.lastSeen = time.Now()

	return entry.limiter
}

// cleanupLoop removes IPs that haven't been seen for 3 minutes
func (i *IPRateLimiter) cleanupLoop() {
	for {
		time.Sleep(1 * time.Minute)

		i.mu.Lock()
		for ip, client := range i.ips {
			if time.Since(client.lastSeen) > 3*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

func peerIsTrustedProxy(remoteAddr string) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	// Caddy in this deployment reverse-proxies to the API on loopback.
	// Only that peer may supply a client address.
	return ip != nil && ip.IsLoopback()
}

func lastForwardedIP(xff string) string {
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}
	return ""
}

func getRealIP(r *http.Request) string {
	remoteHost := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remoteHost = h
	}

	// Trust X-Forwarded-For only from the in-container proxy. Caddy appends the
	// peer it observed, so the last hop is that address. A client-supplied
	// prefix, or X-Real-Ip on a path that does not overwrite it, is ignored.
	if peerIsTrustedProxy(r.RemoteAddr) {
		if ip := lastForwardedIP(r.Header.Get("X-Forwarded-For")); ip != "" {
			return ip
		}
	}

	return remoteHost
}

func (i *IPRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getRealIP(r)

		limiter := i.getLimiter(ip)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
