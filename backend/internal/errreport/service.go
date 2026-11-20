// Package errreport sends the API's unexpected errors to Sentry or GlitchTip,
// when an admin has turned that on. See README.md.
package errreport

import (
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

// Options configure a Service.
type Options struct {
	// Release is the product version events carry (APP_VERSION).
	Release string
}

// Service owns the settings row and the running Sentry client.
type Service struct {
	db   *sql.DB
	opts Options

	// hub is this service's own, so nothing else in the process (or another
	// test) shares its client.
	hub *sentry.Hub

	mu      sync.RWMutex
	current Settings
}

// NewService loads the stored settings and starts reporting if they are on.
func NewService(db *sql.DB, opts Options) (*Service, error) {
	s := &Service{db: db, opts: opts, hub: sentry.NewHub(nil, sentry.NewScope())}
	st, err := load(db)
	if err != nil {
		return nil, err
	}
	if err := s.apply(st); err != nil {
		// A row that no longer parses (it was valid when saved) must not stop
		// the server: report nothing and say why.
		log.Printf("WARNING: error reporting stays off: %v", err)
	}
	return s, nil
}

// Close flushes queued events and stops the running client.
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c := s.hub.Client(); c != nil {
		c.Flush(2 * time.Second)
		c.Close()
		s.hub.BindClient(nil)
	}
}

// Get returns the stored settings, or the defaults when none are stored.
func (s *Service) Get() (Settings, error) {
	return load(s.db)
}

// PublicConfig is what GET /api/error-reporting/config returns. The DSN is
// only included while reporting is on.
type PublicConfig struct {
	Enabled     bool   `json:"enabled"`
	DSN         string `json:"dsn,omitempty"`
	Environment string `json:"environment,omitempty"`
	Release     string `json:"release,omitempty"`
	Browser     bool   `json:"browser,omitempty"`
}

// PublicConfig returns the configuration the SvelteKit server and browsers
// report with.
func (s *Service) PublicConfig() PublicConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.current.Enabled {
		return PublicConfig{}
	}
	return PublicConfig{
		Enabled:     true,
		DSN:         s.current.DSN,
		Environment: s.current.Environment,
		Release:     s.opts.Release,
		Browser:     s.current.BrowserEnabled,
	}
}

// SaveInput is what an admin submits.
type SaveInput struct {
	Enabled        bool
	DSN            string
	Environment    string
	BrowserEnabled bool
}

// Save validates and stores the settings, then applies them to the running
// client, so they take effect without a restart.
func (s *Service) Save(in SaveInput) (Settings, error) {
	st := Settings{
		Enabled:        in.Enabled,
		DSN:            strings.TrimSpace(in.DSN),
		Environment:    strings.TrimSpace(in.Environment),
		BrowserEnabled: in.BrowserEnabled,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if err := validate(st); err != nil {
		return Settings{}, err
	}
	if err := writeRow(s.db, st); err != nil {
		return Settings{}, err
	}
	if err := s.apply(st); err != nil {
		return Settings{}, err
	}
	return st, nil
}

// apply makes st the running configuration and binds a client for it.
func (s *Service) apply(st Settings) error {
	client, err := s.newClient(st)
	if err != nil {
		st.Enabled = false
		client = nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = st
	s.bind(client)
	return err
}
