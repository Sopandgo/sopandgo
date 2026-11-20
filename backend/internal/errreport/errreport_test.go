package errreport_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	"github.com/sopandgo/sopandgo/backend/internal/errreport"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func newService(t *testing.T) *errreport.Service {
	t.Helper()
	store, err := storage.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.DB.Close() })
	svc, err := errreport.NewService(store.DB, errreport.Options{Release: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	return svc
}

func TestFreshInstanceHasReportingOff(t *testing.T) {
	svc := newService(t)

	st, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if st.Enabled || st.DSN != "" || st.BrowserEnabled || st.Environment != "production" {
		t.Fatalf("defaults = %+v", st)
	}
	if got := svc.PublicConfig(); got.Enabled || got.DSN != "" {
		t.Fatalf("public config = %+v, want disabled without a DSN", got)
	}
}

const testDSN = "https://abc123@glitchtip.example.org/7"

func TestSavedSettingsAreReadBackAndPublished(t *testing.T) {
	svc := newService(t)

	saved, err := svc.Save(errreport.SaveInput{
		Enabled: true, DSN: " " + testDSN + " ", Environment: "lab-a", BrowserEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.UpdatedAt == "" {
		t.Fatal("updated_at not set")
	}
	got, err := svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got != saved || !got.Enabled || got.DSN != testDSN || got.Environment != "lab-a" || !got.BrowserEnabled {
		t.Fatalf("got %+v, saved %+v", got, saved)
	}
	want := errreport.PublicConfig{Enabled: true, DSN: testDSN, Environment: "lab-a", Release: "1.2.3", Browser: true}
	if pc := svc.PublicConfig(); pc != want {
		t.Fatalf("public config = %+v, want %+v", pc, want)
	}

	if _, err := svc.Save(errreport.SaveInput{Enabled: false, DSN: testDSN, Environment: "lab-a"}); err != nil {
		t.Fatal(err)
	}
	if pc := svc.PublicConfig(); pc.Enabled || pc.DSN != "" {
		t.Fatalf("after switching off, public config = %+v", pc)
	}
}

func TestSaveRejectsInvalidSettings(t *testing.T) {
	cases := []struct {
		name string
		in   errreport.SaveInput
		want error
	}{
		{"on without a DSN", errreport.SaveInput{Enabled: true, Environment: "production"}, errreport.ErrDSNRequired},
		{"not a DSN", errreport.SaveInput{DSN: "glitchtip.example.org", Environment: "production"}, errreport.ErrDSNInvalid},
		{"no public key", errreport.SaveInput{DSN: "https://glitchtip.example.org/7", Environment: "production"}, errreport.ErrDSNInvalid},
		{"no project", errreport.SaveInput{DSN: "https://abc@glitchtip.example.org/", Environment: "production"}, errreport.ErrDSNInvalid},
		{"other scheme", errreport.SaveInput{DSN: "ftp://abc@glitchtip.example.org/7", Environment: "production"}, errreport.ErrDSNInvalid},
		{"empty environment", errreport.SaveInput{DSN: testDSN, Environment: ""}, errreport.ErrEnvironmentInvalid},
		{"environment with spaces", errreport.SaveInput{DSN: testDSN, Environment: "lab a"}, errreport.ErrEnvironmentInvalid},
		{"environment too long", errreport.SaveInput{DSN: testDSN, Environment: strings.Repeat("a", 65)}, errreport.ErrEnvironmentInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService(t)
			_, err := svc.Save(tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if kind, ok := errkind.KindOf(err); !ok || kind != errkind.Invalid {
				t.Fatalf("kind = %v, want Invalid", kind)
			}
			if st, _ := svc.Get(); st.UpdatedAt != "" {
				t.Fatalf("a rejected save was stored: %+v", st)
			}
		})
	}
}

func TestSaveAcceptsSelfHostedHTTPAndAnEmptyDSNWhileOff(t *testing.T) {
	svc := newService(t)
	for _, in := range []errreport.SaveInput{
		{Enabled: true, DSN: "http://abc@glitchtip.lab.internal:8000/3", Environment: "staging_1.x"},
		{Enabled: false, DSN: "", Environment: "production"},
	} {
		if _, err := svc.Save(in); err != nil {
			t.Fatalf("Save(%+v) = %v", in, err)
		}
	}
}

// enabledService is a Service reporting to a fresh ingest endpoint.
func enabledService(t *testing.T) (*errreport.Service, *ingest) {
	t.Helper()
	in := newIngest(t)
	svc := newService(t)
	if _, err := svc.Save(errreport.SaveInput{Enabled: true, DSN: in.dsn(), Environment: "lab-a"}); err != nil {
		t.Fatal(err)
	}
	return svc, in
}

// serve runs one request carrying personal data through the middleware.
func serve(t *testing.T, h http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/sops/sop-secret-id?q=confidential-search", strings.NewReader("body-secret"))
	req.RemoteAddr = "203.0.113.77:5555"
	req.Header.Set("Authorization", "Bearer token-secret")
	req.Header.Set("Cookie", "refresh_token=cookie-secret")
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	req.Header.Set("User-Agent", "agent-secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func router(h http.HandlerFunc) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/sops/{sopID}", h)
	return mux
}

var personalData = []string{
	"sop-secret-id", "confidential-search", "body-secret", "203.0.113.77", "198.51.100.9",
	"token-secret", "cookie-secret", "agent-secret",
}

func assertNoPersonalData(t *testing.T, raw string, ev map[string]any) {
	t.Helper()
	for _, s := range personalData {
		if strings.Contains(raw, s) {
			t.Errorf("event contains %q: %s", s, raw)
		}
	}
	for _, key := range []string{"request", "user", "breadcrumbs"} {
		if v, ok := ev[key]; ok && v != nil {
			if m, isMap := v.(map[string]any); !isMap || len(m) > 0 {
				t.Errorf("event has %s: %v", key, v)
			}
		}
	}
	if ev["server_name"] != "sopandgo" {
		t.Errorf("server_name = %v, want sopandgo", ev["server_name"])
	}
}

func TestPanicIsAnswered500AndReportedWithoutPersonalData(t *testing.T) {
	svc, in := enabledService(t)
	h := svc.Middleware(router(func(w http.ResponseWriter, r *http.Request) {
		panic("index out of range in sop " + r.PathValue("sopID"))
	}))

	rec := serve(t, h)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), `"internal"`) {
		t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
	}
	svc.Flush(2 * time.Second)

	events, raw := in.received()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	ev := events[0]
	tags, _ := ev["tags"].(map[string]any)
	if tags["route"] != "GET /api/sops/{sopID}" || tags["method"] != "GET" {
		t.Errorf("tags = %v", tags)
	}
	if ev["release"] != "1.2.3" || ev["environment"] != "lab-a" {
		t.Errorf("release/environment = %v/%v", ev["release"], ev["environment"])
	}
	if !strings.Contains(raw[0], "index out of range") {
		t.Errorf("panic message missing: %s", raw[0])
	}
	assertNoPersonalData(t, strings.ReplaceAll(raw[0], "index out of range in sop sop-secret-id", "index out of range"), ev)
}

func TestServerErrorResponsesAreReportedByRouteAndStatus(t *testing.T) {
	for _, status := range []int{500, 504} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			svc, in := enabledService(t)
			h := svc.Middleware(router(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "failed to load sop sop-secret-id", status)
			}))
			if rec := serve(t, h); rec.Code != status {
				t.Fatalf("status = %d", rec.Code)
			}
			svc.Flush(2 * time.Second)
			events, raw := in.received()
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			want := "HTTP " + strconv.Itoa(status) + " GET /api/sops/{sopID}"
			if events[0]["message"] != want {
				t.Errorf("message = %v, want %q", events[0]["message"], want)
			}
			tags, _ := events[0]["tags"].(map[string]any)
			if tags["status"] != strconv.Itoa(status) {
				t.Errorf("tags = %v", tags)
			}
			assertNoPersonalData(t, raw[0], events[0])
		})
	}
}

func TestDeliberateAndClientStatusesAreNotReported(t *testing.T) {
	svc, in := enabledService(t)
	for _, status := range []int{200, 204, 400, 403, 404, 409, 502, 503} {
		h := svc.Middleware(router(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		serve(t, h)
	}
	svc.Flush(2 * time.Second)
	if events, _ := in.received(); len(events) != 0 {
		t.Fatalf("got %d events, want none: %v", len(events), events)
	}
}

func TestNothingIsSentAfterReportingIsSwitchedOff(t *testing.T) {
	svc, in := enabledService(t)
	if _, err := svc.Save(errreport.SaveInput{Enabled: false, DSN: in.dsn(), Environment: "lab-a"}); err != nil {
		t.Fatal(err)
	}
	h := svc.Middleware(router(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	if rec := serve(t, h); rec.Code != http.StatusInternalServerError {
		t.Fatalf("a panic must still be answered 500 while reporting is off, got %d", rec.Code)
	}
	svc.Flush(2 * time.Second)
	if events, _ := in.received(); len(events) != 0 {
		t.Fatalf("got %d events after switching off", len(events))
	}
}

func TestAbortHandlerPanicsPassThrough(t *testing.T) {
	svc := newService(t)
	h := svc.Middleware(router(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	serve(t, h)
}

func TestTestEventIsDeliveredWithTheSavedSettings(t *testing.T) {
	svc, in := enabledService(t)

	id, err := svc.SendTest()
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("no event id")
	}
	events, _ := in.received()
	if len(events) != 1 || events[0]["message"] != "SOPandGO test event" || events[0]["event_id"] != id {
		t.Fatalf("events = %v, id %s", events, id)
	}
}

func TestTestEventNeedsReportingOn(t *testing.T) {
	svc := newService(t)
	if _, err := svc.SendTest(); !errors.Is(err, errreport.ErrNotEnabled) {
		t.Fatalf("err = %v, want ErrNotEnabled", err)
	}
}

func TestTestEventSaysWhenDeliveryFails(t *testing.T) {
	svc, in := enabledService(t)
	in.setFail(true)
	if _, err := svc.SendTest(); !errors.Is(err, errreport.ErrDeliveryFailed) {
		t.Fatalf("err = %v, want ErrDeliveryFailed", err)
	}
}
