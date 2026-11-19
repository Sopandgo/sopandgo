package testenv

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/api"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/branding"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/objectstore"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// Env holds all your initialized services for testing.
type Env struct {
	Store        *storage.Storage
	AuditLogger  *audit.Logger
	AuthService  *auth.Service
	SOPService   *sop.Service
	MailService  *mail.Service
	SMTPSettings *mail.SMTPSettingsStore
	Branding     *branding.Service
	// Notify fans out to the integrations saved in IntegrationSettings.
	Notify              *notify.Service
	IntegrationSettings *notify.SettingsStore
	API                 *api.Server
	MockMailSender      *MockMailSender
	// ObjectStore is the bucket "Back up now" and the connection test reach.
	ObjectStore *objectstore.Memory
	DataDir     string
	Ctx         context.Context
}

// Option changes how New builds the environment.
type Option func(*options)

type options struct {
	demoMode bool
}

// WithDemoMode builds the API with DEMO_MODE on.
func WithDemoMode() Option {
	return func(o *options) { o.demoMode = true }
}

func New(t *testing.T, opts ...Option) *Env {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dataDir := t.TempDir()

	store, err := storage.OpenInMemory()
	if err != nil {
		t.Fatalf("failed to initialize in-memory storage: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("failed to initialize audit logger: %v", err)
	}

	// Fixed 32-byte key so 2FA and admin settings endpoints can be exercised in tests
	testEncKey := []byte("01234567890123456789012345678901")

	authService := auth.NewService(store.DB, auditLogger, dataDir, testEncKey)

	authService.StartSessionsCleanupTask(ctx, 24*time.Hour)

	sopService := sop.NewService(store.DB, auditLogger, dataDir, "test-generator", false, nil)

	// Initialize the exported mock (API tests do not send real SMTP)
	mockSender := &MockMailSender{}
	mailService, err := mail.NewService(mockSender, auditLogger)
	if err != nil {
		t.Fatalf("failed to initialize mail service: %v", err)
	}

	smtpSettingsStore := mail.NewSMTPSettingsStore(store.DB, testEncKey)
	// Email is only used with a saved transport (otherwise manual links); the mock
	// sender does the sending, this row just makes SMTP count as configured.
	if err := smtpSettingsStore.Save(mail.SaveSMTPInput{
		Host:        "smtp.testenv.local",
		Port:        "587",
		Username:    "testenv",
		FromAddress: "noreply@testenv.local",
		Password:    "testenv",
	}); err != nil {
		t.Fatalf("failed to save test SMTP settings: %v", err)
	}
	integrationSettings := notify.NewSettingsStore(store.DB, testEncKey)
	notifyService := notify.NewService(integrationSettings, auditLogger)

	config := api.Config{
		Origin:   "http://testenv.local",
		DemoMode: o.demoMode,
	}
	backupService := backup.NewService(store.DB, dataDir, "test", storage.LatestSchemaVersion())

	s3Settings := backup.NewS3SettingsStore(store.DB, testEncKey)
	// Never started: API tests run "Back up now" against the in-memory bucket.
	objectStore := objectstore.NewMemory("lab")
	s3Scheduler := backup.NewS3Scheduler(backupService, s3Settings, objectStore.Opener(), auditLogger)

	brandingService := branding.NewService(store.DB, auditLogger)

	apiServer := api.New(config, auditLogger, authService, sopService, mailService, smtpSettingsStore, notifyService, integrationSettings, backupService, s3Settings, s3Scheduler, brandingService)

	return &Env{
		Store:               store,
		AuditLogger:         auditLogger,
		AuthService:         authService,
		SOPService:          sopService,
		MailService:         mailService,
		SMTPSettings:        smtpSettingsStore,
		Branding:            brandingService,
		Notify:              notifyService,
		IntegrationSettings: integrationSettings,
		API:                 apiServer,
		MockMailSender:      mockSender,
		ObjectStore:         objectStore,
		DataDir:             dataDir,
		Ctx:                 ctx,
	}
}

// SeedTestUser injects a dummy user directly into the database to satisfy
// Foreign Key constraints during testing without triggering audit logs.
func (e *Env) SeedTestUser(t *testing.T, userID, email, role string) {
	query := `
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at) 
		VALUES (?, 'Test User', ?, 'dummy-hash', ?, 1, '2026-01-01T00:00:00Z')
	`
	_, err := e.Store.DB.Exec(query, userID, email, role)
	if err != nil && !storage.IsUniqueViolation(err) {
		t.Fatalf("Failed to seed test user: %v", err)
	}
}

// --- EXPORTED MOCK MAIL SENDER ---
type MockMailSender struct {
	SentCount   int
	LastTo      []string
	LastSubject string
	FailErr     error
}

// Note: 'to' is now correctly typed as []string
func (m *MockMailSender) Send(to []string, subject, body string) error {
	if m.FailErr != nil {
		return m.FailErr
	}
	m.SentCount++
	m.LastTo = to
	m.LastSubject = subject
	return nil
}

func (m *MockMailSender) SetFail(v bool) {
	if v {
		m.FailErr = errors.New("smtp send failed in test")
		return
	}
	m.FailErr = nil
}
