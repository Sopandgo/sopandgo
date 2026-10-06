package api

import (
	"net/http"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"golang.org/x/time/rate"
)

type Config struct {
	Origin string // e.g. "http://localhost:8087" or "https://myapp.com"
}

type Server struct {
	Config              Config
	auditLogger         *audit.Logger
	authService         *auth.Service
	sopService          *sop.Service
	mailService         *mail.Service
	smtpSettings        *mail.SMTPSettingsStore
	notifyService       *notify.Service
	integrationSettings *notify.SettingsStore
	backupSvc           *backup.Service
	s3Backup            *backup.S3Scheduler
	mux                 *http.ServeMux
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func New(
	config Config,
	auditLogger *audit.Logger,
	authService *auth.Service,
	sopService *sop.Service,
	mailService *mail.Service,
	smtpSettings *mail.SMTPSettingsStore,
	notifyService *notify.Service,
	integrationSettings *notify.SettingsStore,
	backupSvc *backup.Service,
	s3Backup *backup.S3Scheduler,
) *Server {
	s := &Server{
		Config:              config,
		auditLogger:         auditLogger,
		authService:         authService,
		sopService:          sopService,
		mailService:         mailService,
		smtpSettings:        smtpSettings,
		notifyService:       notifyService,
		integrationSettings: integrationSettings,
		backupSvc:           backupSvc,
		s3Backup:            s3Backup,
		mux:                 http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	loginLimiter := NewIPRateLimiter(rate.Every(12*time.Second), 3)
	generalLimiter := NewIPRateLimiter(20, 10)

	// health
	s.mux.HandleFunc("GET /api/health", generalLimiter.Middleware(s.handleHealth))

	// auth
	s.mux.HandleFunc("POST /api/auth/login", loginLimiter.Middleware(s.handleLogin))
	s.mux.HandleFunc("POST /api/auth/refresh", generalLimiter.Middleware(s.handleRefresh))
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("PATCH /api/auth/reset-password", loginLimiter.Middleware(s.handleResetPassword))

	// protected routes
	protected := func(h http.HandlerFunc) http.HandlerFunc {
		// Order: RateLimit -> Auth Check -> Password change guard -> Maintenance -> Handler
		return generalLimiter.Middleware(s.withAuth(s.withPasswordChangeGuard(s.withMaintenanceGuard(h))))
	}

	// user
	s.mux.HandleFunc("PATCH /api/auth/me/update-password", protected(s.handleUpdatePassword))
	s.mux.HandleFunc("PATCH /api/auth/me/locale", protected(s.handleUpdateMyLocale))
	s.mux.HandleFunc("GET /api/auth/me", protected(s.handleGetMe))
	s.mux.HandleFunc("GET /api/auth/me/signature-status", protected(s.handleGetMySignatureStatus))

	// sops
	s.mux.HandleFunc("POST /api/sops", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleRegisterSOP)),
	)
	s.mux.HandleFunc("GET /api/sops", protected(s.handleListSOPs))
	s.mux.HandleFunc("GET /api/sops/{sopID}", protected(s.handleGetSOPByID))
	s.mux.HandleFunc("POST /api/sops/{sopID}/favorite", protected(s.handleFavoriteSOP))
	s.mux.HandleFunc("DELETE /api/sops/{sopID}/favorite", protected(s.handleUnfavoriteSOP))

	// tags (Global)
	s.mux.HandleFunc("POST /api/tags", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleCreateTag)),
	)
	s.mux.HandleFunc("GET /api/tags", protected(s.handleListTags))
	s.mux.HandleFunc("PATCH /api/tags/{tagID}/status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleSetTagStatus)), // Admin only to hide global tags
	)

	// sop tags (Attachment)
	s.mux.HandleFunc("POST /api/sops/{sopID}/tags/{tagID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleAttachTagToSOP)),
	)
	s.mux.HandleFunc("DELETE /api/sops/{sopID}/tags/{tagID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleDetachTagFromSOP)),
	)

	// sop versions
	s.mux.HandleFunc("POST /api/sops/{sopID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleRegisterSOPVersion)),
	) // POST {sop_ID, content}
	s.mux.HandleFunc("GET /api/sops/{sopID}/version-latest", protected(s.handleGetSOPVersionSummaryLatest))

	s.mux.HandleFunc("GET /api/sops/{sopID}/versions", protected(s.handleListSOPVersions))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}", protected(s.handleGetSOPVersionByID))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/summary", protected(s.handleGetSOPVersionSummaryByID))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/download", protected(s.handleDownloadSOPVersion))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/download-pdf", protected(s.handleDownloadSOPVersionPDF))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/integrity", protected(s.handleCheckVersionIntegrity))
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/diff", protected(s.handleDiffSOPVersion))
	s.mux.HandleFunc("GET /api/activity/publishes", protected(s.handleListRecentPublishes))
	s.mux.HandleFunc("GET /api/training/coverage", protected(
		s.requireScope(auth.ScopeTrainingRead, s.handleTrainingCoverage),
	))
	s.mux.HandleFunc("GET /api/sops/{sopID}/training", protected(
		s.requireScope(auth.ScopeTrainingRead, s.handleSOPTrainingCoverage),
	))

	// sop version lifecycle
	// sop version states (Lifecycle)
	s.mux.HandleFunc("POST /api/sops/{sopID}/versions/{sopVersionID}/promote", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handlePromoteSOPVersion)),
	)
	s.mux.HandleFunc("POST /api/sops/{sopID}/versions/{sopVersionID}/approve", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleApproveSOPVersion)),
	)
	s.mux.HandleFunc("POST /api/sops/{sopID}/versions/{sopVersionID}/reject", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleRejectSOPVersion)),
	)

	//assets
	s.mux.HandleFunc("POST /api/sops/{sopID}/assets", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleAddAsset)),
	) // Multipart Form
	s.mux.HandleFunc("GET /api/sops/{sopID}/assets", protected(s.handleListAssets))
	s.mux.HandleFunc("GET /api/sops/{sopID}/assets/{assetID}", protected(s.handleGetAsset))
	s.mux.HandleFunc("GET /api/sops/{sopID}/assets/{assetID}/download", protected(s.handleDownloadAsset))
	s.mux.HandleFunc("GET /api/sops/{sopID}/assets/{assetID}/integrity", protected(s.handleCheckAssetIntegrity))

	// acknowledgments
	// Author acks are created with the version. Approver acks are created by approve.
	s.mux.HandleFunc("POST /api/sops/{sopID}/versions/{sopVersionID}/add-reader", protected(
		s.requireScope(auth.ScopeSOPSignReader, s.handleAddAcknowledgmentReader)),
	)
	s.mux.HandleFunc("GET /api/sops/{sopID}/versions/{sopVersionID}/acks", protected(s.handleListVersionAcknowledgments))

	// admin/users
	s.mux.HandleFunc("GET /api/admin/users", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminListUsers)),
	)
	s.mux.HandleFunc("POST /api/admin/users/register", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRegisterUser)),
	)
	s.mux.HandleFunc("POST /api/admin/users/{userID}/reset-password", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminTriggerPasswordReset),
	))
	s.mux.HandleFunc("PATCH /api/admin/users/{userID}/update-role", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminUpdateRole)),
	)
	s.mux.HandleFunc("PATCH /api/admin/users/{userID}/update-status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminUpdateStatus)),
	)
	s.mux.HandleFunc("GET /api/admin/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminListSessions)),
	)
	s.mux.HandleFunc("DELETE /api/admin/users/{userID}/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRevokeUserSessions)),
	)
	s.mux.HandleFunc("DELETE /api/admin/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRevokeAllSessions)),
	)
	// admin/autit-logs
	s.mux.HandleFunc("GET /api/admin/audit-logs", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminListAuditLogs)),
	)
	s.mux.HandleFunc("GET /api/admin/audit-logs/options", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminAuditFilterOptions)),
	)
	//admin/integrity
	s.mux.HandleFunc("GET /api/admin/integrity", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminIntegrity)),
	)

	// admin / email (SMTP + optional Resend)
	s.mux.HandleFunc("GET /api/admin/settings/email", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetSMTPSettings)),
	)
	s.mux.HandleFunc("GET /api/admin/settings/smtp", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetSMTPSettings)),
	)
	s.mux.HandleFunc("PUT /api/admin/settings/smtp", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutSMTPSettings)),
	)
	s.mux.HandleFunc("POST /api/admin/settings/smtp/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostSMTPTest)),
	)
	s.mux.HandleFunc("PATCH /api/admin/settings/mail-transport", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchMailTransport)),
	)
	s.mux.HandleFunc("PUT /api/admin/settings/resend", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutResendSettings)),
	)
	s.mux.HandleFunc("PATCH /api/admin/settings/mail-mode", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchMailMode)),
	)
	s.mux.HandleFunc("PATCH /api/admin/settings/default-locale", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchDefaultLocale)),
	)
	// admin / integrations (Slack, Gotify, generic webhook)
	s.mux.HandleFunc("GET /api/admin/settings/integrations", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetIntegrations)),
	)
	s.mux.HandleFunc("PUT /api/admin/settings/integrations/slack", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutSlackIntegration)),
	)
	s.mux.HandleFunc("PUT /api/admin/settings/integrations/gotify", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutGotifyIntegration)),
	)
	s.mux.HandleFunc("PUT /api/admin/settings/integrations/webhook", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutWebhookIntegration)),
	)
	s.mux.HandleFunc("POST /api/admin/settings/integrations/{channel}/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostIntegrationTest)),
	)
	// admin / backups
	s.mux.HandleFunc("GET /api/admin/backups/status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupStatus)),
	)
	s.mux.HandleFunc("GET /api/admin/backups/export", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupExport)),
	)
	s.mux.HandleFunc("POST /api/admin/backups/import", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupImport)),
	)
	s.mux.HandleFunc("POST /api/admin/backups/apply", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupApply)),
	)
}

func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}
