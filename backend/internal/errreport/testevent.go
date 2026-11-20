package errreport

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

var (
	// ErrNotEnabled: a test event needs saved settings with reporting on.
	ErrNotEnabled = errkind.New(errkind.Conflict, "error reporting is off")
	// ErrDeliveryFailed: the endpoint could not be reached or refused the event.
	ErrDeliveryFailed = errors.New("the test event was not delivered")
)

// testEventMessage is what the test event says in Sentry or GlitchTip.
const testEventMessage = "SOPandGO test event"

// SendTest sends one message event with the saved settings and waits for the
// endpoint's answer, so the admin learns whether the DSN works.
func (s *Service) SendTest() (string, error) {
	s.mu.RLock()
	st := s.current
	s.mu.RUnlock()
	if !st.Enabled {
		return "", ErrNotEnabled
	}

	rt := &recordingRoundTripper{base: http.DefaultTransport}
	transport := sentry.NewHTTPSyncTransport()
	transport.Timeout = 5 * time.Second
	opts := s.clientOptions(st)
	opts.Transport = transport
	opts.HTTPTransport = rt
	client, err := sentry.NewClient(opts)
	if err != nil {
		return "", ErrDeliveryFailed
	}
	defer client.Close()

	id := sentry.NewHub(client, sentry.NewScope()).CaptureMessage(testEventMessage)
	if id == nil || !rt.succeeded() {
		return "", ErrDeliveryFailed
	}
	return string(*id), nil
}

// recordingRoundTripper remembers whether the last request got a 2xx answer.
type recordingRoundTripper struct {
	base http.RoundTripper
	mu   sync.Mutex
	ok   bool
}

func (t *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	t.mu.Lock()
	t.ok = err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300
	t.mu.Unlock()
	return resp, err
}

func (t *recordingRoundTripper) succeeded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ok
}
