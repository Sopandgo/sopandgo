package errreport

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// serverName replaces the container hostname in every event.
const serverName = "sopandgo"

// newClient builds the client for st, or nil when reporting is off.
func (s *Service) newClient(st Settings) (*sentry.Client, error) {
	if !st.Enabled || st.DSN == "" {
		return nil, nil
	}
	return sentry.NewClient(s.clientOptions(st))
}

func (s *Service) clientOptions(st Settings) sentry.ClientOptions {
	return sentry.ClientOptions{
		Dsn:              st.DSN,
		Release:          s.opts.Release,
		Environment:      st.Environment,
		ServerName:       serverName,
		AttachStacktrace: true,
		MaxBreadcrumbs:   -1,
		SendDefaultPII:   false,
		BeforeSend:       scrub,
	}
}

// keptContexts are the only event contexts sent: what the Go runtime and OS
// are, and the trace IDs Sentry groups by.
var keptContexts = map[string]bool{"os": true, "runtime": true, "device": true, "trace": true}

// scrub is the last step before an event leaves: whatever an integration or
// a future caller attached, no request, user, breadcrumb, attachment or other
// context goes out, and the host name is replaced.
func scrub(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.Request = nil
	event.User = sentry.User{}
	event.Breadcrumbs = nil
	event.Attachments = nil
	for name := range event.Contexts {
		if !keptContexts[name] {
			delete(event.Contexts, name)
		}
	}
	event.ServerName = serverName
	return event
}

// bind swaps the running client. The old one is flushed and closed in the
// background so a save does not wait on the network.
func (s *Service) bind(client *sentry.Client) {
	old := s.hub.Client()
	s.hub.BindClient(client)
	if old != nil && old != client {
		go func() {
			old.Flush(2 * time.Second)
			old.Close()
		}()
	}
}

// Flush waits up to timeout for queued events to be sent.
func (s *Service) Flush(timeout time.Duration) bool {
	return s.hub.Flush(timeout)
}

// Middleware recovers panics, answering 500, and reports panics and 5xx
// responses while reporting is on, except two the API answers on purpose:
// 502 when a service the admin configured fails (SMTP, S3, Slack, the TSA,
// a test button) and 503 for the maintenance lock and an unavailable second
// factor. Those are the admin's to fix, not bugs. Events name the route
// pattern only, never the URL, so IDs and queries stay out.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			log.Printf("panic serving %s: %v\n%s", routeOf(r), v, debug.Stack())
			hub := s.hub.Clone()
			hub.ConfigureScope(func(sc *sentry.Scope) { sc.SetTags(tags(r, http.StatusInternalServerError)) })
			hub.Recover(v)
			if !sw.wroteHeader {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal", "detail": "internal server error"})
			}
		}()
		next.ServeHTTP(sw, r)
		if reportedStatus(sw.status) {
			s.captureStatus(r, sw.status)
		}
	})
}

func reportedStatus(status int) bool {
	return status >= 500 && status != http.StatusBadGateway && status != http.StatusServiceUnavailable
}

func (s *Service) captureStatus(r *http.Request, status int) {
	hub := s.hub.Clone()
	if hub.Client() == nil {
		return
	}
	route := routeOf(r)
	hub.WithScope(func(sc *sentry.Scope) {
		sc.SetTags(tags(r, status))
		sc.SetFingerprint([]string{r.Method, route, strconv.Itoa(status)})
		sc.SetLevel(sentry.LevelError)
		hub.CaptureMessage(fmt.Sprintf("HTTP %d %s", status, route))
	})
}

// routeOf is the matched pattern with its method, such as
// "GET /api/sops/{sopID}". The mux sets r.Pattern on the request it was
// given, so it is known here after routing.
func routeOf(r *http.Request) string {
	p := r.Pattern
	if p == "" {
		return r.Method + " (unmatched)"
	}
	if !strings.HasPrefix(p, r.Method+" ") && !strings.Contains(p, " ") {
		return r.Method + " " + p
	}
	return p
}

func tags(r *http.Request, status int) map[string]string {
	return map[string]string{
		"method": r.Method,
		"route":  routeOf(r),
		"status": strconv.Itoa(status),
	}
}

// statusWriter records the status a handler answered with.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
