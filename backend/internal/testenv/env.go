package testenv

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/api"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// Env holds all your initialized services for testing.
type Env struct {
	Store          *storage.Storage
	AuditLogger    *audit.Logger
	AuthService    *auth.Service
	SOPService     *sop.Service
	MailService    *mail.Service
	SMTPSettings   *mail.SMTPSettingsStore
	API            *api.Server
	MockMailSender *MockMailSender
	DataDir        string
	Ctx            context.Context
}

func New(t *testing.T) *Env {
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

	authService := auth.NewService(store.DB, auditLogger)

	authService.StartSessionsCleanupTask(ctx, 24*time.Hour)

	sopService := sop.NewService(store.DB, auditLogger, dataDir, "test-generator", false, nil)

	// Initialize the exported mock (API tests do not send real SMTP)
	mockSender := &MockMailSender{}
	mailService, err := mail.NewService(mockSender, auditLogger)
	if err != nil {
		t.Fatalf("failed to initialize mail service: %v", err)
	}

	// Fixed 32-byte key so admin SMTP settings endpoints can be exercised in tests
	testEncKey := []byte("01234567890123456789012345678901")
	smtpSettingsStore := mail.NewSMTPSettingsStore(store.DB, testEncKey)
	integrationSettings := notify.NewSettingsStore(store.DB, testEncKey)
	notifyService := notify.NewService(integrationSettings, auditLogger)

	config := api.Config{
		Origin: "http://testenv.local",
	}
	backupService := backup.NewService(store.DB, dataDir, "test", storage.LatestSchemaVersion())

	apiServer := api.New(config, auditLogger, authService, sopService, mailService, smtpSettingsStore, notifyService, integrationSettings, backupService, nil)

	return &Env{
		Store:          store,
		AuditLogger:    auditLogger,
		AuthService:    authService,
		SOPService:     sopService,
		MailService:    mailService,
		SMTPSettings:   smtpSettingsStore,
		API:            apiServer,
		MockMailSender: mockSender,
		DataDir:        dataDir,
		Ctx:            ctx,
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
	if err != nil && !strings.Contains(err.Error(), "UNIQUE constraint failed") {
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
