package api

import (
	"log"
	"net/http"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/anchor"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/branding"
	"github.com/sopandgo/sopandgo/backend/internal/errreport"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"golang.org/x/time/rate"
)

type Config struct {
	Origin string // e.g. "http://localhost:8087" or "https://myapp.com"
	// DemoMode makes the instance a read-only public demo (DEMO_MODE). The demo
	// guard wraps every route, the demo session endpoint is registered, audit
	// responses are redacted, and mail and notifications are silent.
	DemoMode bool
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
	s3Settings          *backup.S3SettingsStore
	s3Backup            *backup.S3Scheduler
	brandingService     *branding.Service
	anchoring           *anchor.Service    // nil until SetAuditAnchoring
	errReport           *errreport.Service // nil until SetErrorReporting
	// twoFactorLimiter is keyed by user, not IP, so second-factor guesses
	// spread over many addresses still hit a ceiling.
	twoFactorLimiter *IPRateLimiter
	mux              *http.ServeMux
	// handler is what ServeHTTP and Serve run: the mux, wrapped in the demo
	// guard when demo mode is on.
	handler http.Handler
	// routes lists every registered pattern ("METHOD /path"), in order.
	routes []string
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// handle registers h on the mux and records the pattern in the route table.
func (s *Server) handle(pattern string, h http.HandlerFunc) {
	s.routes = append(s.routes, pattern)
	s.mux.HandleFunc(pattern, h)
}

// Routes returns every registered route pattern, such as
// "GET /api/sops/{sopID}". Tests drive route-wide guarantees from it, so new
// routes are covered without editing a hand-written list.
func (s *Server) Routes() []string {
	return append([]string(nil), s.routes...)
}

func (s *Server) writeAudit(eventType, entityType, entityID string, actorUserID *string, payload any) {
	if s.auditLogger == nil {
		return
	}
	if err := s.auditLogger.Log(nil, eventType, entityType, entityID, actorUserID, payload); err != nil {
		log.Printf("audit write failed for %s/%s/%s: %v", eventType, entityType, entityID, err)
	}
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
	s3Settings *backup.S3SettingsStore,
	s3Backup *backup.S3Scheduler,
	brandingService *branding.Service,
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
		s3Settings:          s3Settings,
		s3Backup:            s3Backup,
		brandingService:     brandingService,
		twoFactorLimiter:    NewIPRateLimiter(rate.Every(time.Minute), 5),
		mux:                 http.NewServeMux(),
	}

	s.registerRoutes()
	s.handler = s.mux
	if config.DemoMode {
		// Nothing leaves a demo: no mail, no Slack, Gotify or webhook calls.
		if mailService != nil {
			mailService.Silence()
		}
		if notifyService != nil {
			notifyService.Silence()
		}
		s.registerDemoRoutes()
		s.handler = withDemoGuard(s.mux)
	}
	return s
}

func (s *Server) registerRoutes() {
	loginLimiter := NewIPRateLimiter(rate.Every(12*time.Second), 3)
	generalLimiter := NewIPRateLimiter(20, 10)

	// health
	s.handle("GET /api/health", generalLimiter.Middleware(s.handleHealth))

	// branding (public: the sign-in page shows the organization's name and logo)
	s.handle("GET /api/branding", generalLimiter.Middleware(s.handleGetBranding))
	s.handle("GET /api/branding/logo", generalLimiter.Middleware(s.handleGetBrandingLogo))

	// error reporting (public: the SvelteKit server and browsers report with it;
	// a DSN only allows sending events, see docs/adr/0004)
	s.handle("GET /api/error-reporting/config", generalLimiter.Middleware(s.handleGetErrorReportingConfig))

	// auth
	s.handle("POST /api/auth/login", loginLimiter.Middleware(s.handleLogin))
	s.handle("POST /api/auth/refresh", generalLimiter.Middleware(s.handleRefresh))
	s.handle("POST /api/auth/logout", s.handleLogout)
	s.handle("PATCH /api/auth/reset-password", loginLimiter.Middleware(s.handleResetPassword))
	// Second sign-in step, after a correct password on an account with 2FA.
	s.handle("POST /api/auth/login/two-factor", loginLimiter.Middleware(s.handleLoginTwoFactor))
	s.handle("POST /api/auth/login/two-factor/email", loginLimiter.Middleware(s.handleLoginTwoFactorEmail))

	// protected routes
	protected := func(h http.HandlerFunc) http.HandlerFunc {
		// Order: RateLimit -> Auth Check -> Password change guard -> Maintenance -> Handler
		return generalLimiter.Middleware(s.withAuth(s.withPasswordChangeGuard(s.withMaintenanceGuard(h))))
	}

	// user
	// Checks the current password, so it shares the login limiter.
	s.handle("PATCH /api/auth/me/update-password", loginLimiter.Middleware(protected(s.handleUpdatePassword)))
	s.handle("POST /api/auth/me/sessions/sign-out-others", protected(s.handleSignOutOtherSessions))
	s.handle("PATCH /api/auth/me/locale", protected(s.handleUpdateMyLocale))
	s.handle("PATCH /api/auth/me/theme", protected(s.handleUpdateMyTheme))
	s.handle("POST /api/auth/me/avatar", protected(s.handleUploadMyAvatar))
	s.handle("DELETE /api/auth/me/avatar", protected(s.handleDeleteMyAvatar))
	s.handle("GET /api/auth/me", protected(s.handleGetMe))
	s.handle("GET /api/auth/me/signature-status", protected(s.handleGetMySignatureStatus))
	// two-factor (self-service). Endpoints that check the password share the login limiter.
	s.handle("GET /api/auth/me/two-factor", protected(s.handleGetMyTwoFactor))
	s.handle("POST /api/auth/me/two-factor/setup", loginLimiter.Middleware(protected(s.handleBeginTwoFactorSetup)))
	s.handle("POST /api/auth/me/two-factor/confirm", loginLimiter.Middleware(protected(s.handleConfirmTwoFactorSetup)))
	s.handle("DELETE /api/auth/me/two-factor", loginLimiter.Middleware(protected(s.handleDisableTwoFactor)))
	s.handle("POST /api/auth/me/two-factor/recovery-codes", loginLimiter.Middleware(protected(s.handleRegenerateRecoveryCodes)))
	s.handle("GET /api/users/{userID}/avatar", protected(s.handleGetUserAvatar))

	// sops
	s.handle("POST /api/sops", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleRegisterSOP)),
	)
	s.handle("GET /api/sops", protected(s.handleListSOPs))
	s.handle("GET /api/sops/{sopID}", protected(s.handleGetSOPByID))
	s.handle("POST /api/sops/{sopID}/favorite", protected(s.handleFavoriteSOP))
	s.handle("DELETE /api/sops/{sopID}/favorite", protected(s.handleUnfavoriteSOP))
	// Approvers decide what readers use, so they also retire SOPs and bring them back
	s.handle("POST /api/sops/{sopID}/archive", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleArchiveSOP)),
	)
	s.handle("DELETE /api/sops/{sopID}/archive", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleUnarchiveSOP)),
	)

	// tags (Global)
	s.handle("POST /api/tags", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleCreateTag)),
	)
	s.handle("GET /api/tags", protected(s.handleListTags))
	s.handle("PATCH /api/tags/{tagID}/status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleSetTagStatus)), // Admin only to hide global tags
	)

	// sop tags (Attachment)
	s.handle("POST /api/sops/{sopID}/tags/{tagID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleAttachTagToSOP)),
	)
	s.handle("DELETE /api/sops/{sopID}/tags/{tagID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleDetachTagFromSOP)),
	)

	// sop versions
	s.handle("POST /api/sops/{sopID}", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleRegisterSOPVersion)),
	) // POST {sop_ID, content}
	s.handle("GET /api/sops/{sopID}/version-latest", protected(s.handleGetSOPVersionSummaryLatest))

	s.handle("GET /api/sops/{sopID}/versions", protected(s.handleListSOPVersions))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}", protected(s.handleGetSOPVersionByID))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/summary", protected(s.handleGetSOPVersionSummaryByID))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/download", protected(s.handleDownloadSOPVersion))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/download-pdf", protected(s.handleDownloadSOPVersionPDF))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/integrity", protected(s.handleCheckVersionIntegrity))
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/diff", protected(s.handleDiffSOPVersion))
	s.handle("GET /api/activity/publishes", protected(s.handleListRecentPublishes))
	s.handle("GET /api/training/coverage", protected(
		s.requireScope(auth.ScopeTrainingRead, s.handleTrainingCoverage),
	))
	s.handle("GET /api/sops/{sopID}/training", protected(
		s.requireScope(auth.ScopeTrainingRead, s.handleSOPTrainingCoverage),
	))

	// sop version lifecycle
	// sop version states (Lifecycle)
	s.handle("POST /api/sops/{sopID}/versions/{sopVersionID}/promote", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handlePromoteSOPVersion)),
	)
	s.handle("POST /api/sops/{sopID}/versions/{sopVersionID}/approve", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleApproveSOPVersion)),
	)
	s.handle("POST /api/sops/{sopID}/versions/{sopVersionID}/reject", protected(
		s.requireScope(auth.ScopeSOPSignApprover, s.handleRejectSOPVersion)),
	)

	//assets
	s.handle("POST /api/sops/{sopID}/assets", protected(
		s.requireScope(auth.ScopeSOPWrite, s.handleAddAsset)),
	) // Multipart Form
	s.handle("GET /api/sops/{sopID}/assets", protected(s.handleListAssets))
	s.handle("GET /api/sops/{sopID}/assets/{assetID}", protected(s.handleGetAsset))
	s.handle("GET /api/sops/{sopID}/assets/{assetID}/download", protected(s.handleDownloadAsset))
	s.handle("GET /api/sops/{sopID}/assets/{assetID}/integrity", protected(s.handleCheckAssetIntegrity))

	// acknowledgments
	// Author acks are created with the version. Approver acks are created by approve.
	s.handle("POST /api/sops/{sopID}/versions/{sopVersionID}/add-reader", protected(
		s.requireScope(auth.ScopeSOPSignReader, s.handleAddAcknowledgmentReader)),
	)
	s.handle("GET /api/sops/{sopID}/versions/{sopVersionID}/acks", protected(s.handleListVersionAcknowledgments))

	// admin/users
	s.handle("GET /api/admin/users", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminListUsers)),
	)
	s.handle("POST /api/admin/users/register", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRegisterUser)),
	)
	s.handle("POST /api/admin/users/{userID}/reset-password", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminTriggerPasswordReset),
	))
	s.handle("PATCH /api/admin/users/{userID}/update-role", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminUpdateRole)),
	)
	s.handle("PATCH /api/admin/users/{userID}/update-status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminUpdateStatus)),
	)
	s.handle("DELETE /api/admin/users/{userID}/two-factor", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminResetUserTwoFactor)),
	)
	s.handle("GET /api/admin/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminListSessions)),
	)
	s.handle("DELETE /api/admin/users/{userID}/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRevokeUserSessions)),
	)
	s.handle("DELETE /api/admin/sessions", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRevokeAllSessions)),
	)
	// admin/audit-logs: the only admin routes the auditor reads (audit:read).
	// Every other /api/admin route needs admin tools.
	s.handle("GET /api/admin/audit-logs", protected(
		s.requireScope(auth.ScopeAuditRead, s.handleAdminListAuditLogs)),
	)
	s.handle("GET /api/admin/audit-logs/options", protected(
		s.requireScope(auth.ScopeAuditRead, s.handleAdminAuditFilterOptions)),
	)
	//admin/integrity
	s.handle("GET /api/admin/integrity", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminIntegrity)),
	)

	// admin/audit-anchoring (external anchoring of the audit chain)
	s.handle("GET /api/admin/audit-anchoring", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetAuditAnchoring)),
	)
	s.handle("PUT /api/admin/audit-anchoring", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutAuditAnchoring)),
	)
	s.handle("POST /api/admin/audit-anchoring/probe", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminProbeAuditAnchoring)),
	)
	s.handle("POST /api/admin/audit-anchoring/run", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminRunAuditAnchoring)),
	)
	s.handle("GET /api/admin/audit-anchoring/bundle", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminAuditEvidenceBundle)),
	)

	// admin / email (SMTP + optional Resend)
	s.handle("GET /api/admin/settings/email", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetSMTPSettings)),
	)
	s.handle("GET /api/admin/settings/smtp", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetSMTPSettings)),
	)
	s.handle("PUT /api/admin/settings/smtp", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutSMTPSettings)),
	)
	s.handle("POST /api/admin/settings/smtp/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostSMTPTest)),
	)
	s.handle("PATCH /api/admin/settings/mail-transport", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchMailTransport)),
	)
	s.handle("PUT /api/admin/settings/resend", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutResendSettings)),
	)
	s.handle("PATCH /api/admin/settings/mail-mode", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchMailMode)),
	)
	s.handle("PATCH /api/admin/settings/default-locale", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchDefaultLocale)),
	)
	// admin / security (organization-wide 2FA requirement)
	s.handle("GET /api/admin/settings/security", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetSecuritySettings)),
	)
	s.handle("PATCH /api/admin/settings/security", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPatchSecuritySettings)),
	)
	// admin / branding (name, logo, primary colour)
	s.handle("PUT /api/admin/settings/branding", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutBranding)),
	)
	s.handle("DELETE /api/admin/settings/branding", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminResetBranding)),
	)
	s.handle("POST /api/admin/settings/branding/logo", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminUploadBrandingLogo)),
	)
	s.handle("DELETE /api/admin/settings/branding/logo", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminDeleteBrandingLogo)),
	)
	// admin / integrations (Slack, Gotify, generic webhook)
	s.handle("GET /api/admin/settings/integrations", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetIntegrations)),
	)
	s.handle("PUT /api/admin/settings/integrations/slack", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutSlackIntegration)),
	)
	s.handle("PUT /api/admin/settings/integrations/gotify", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutGotifyIntegration)),
	)
	s.handle("PUT /api/admin/settings/integrations/webhook", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutWebhookIntegration)),
	)
	s.handle("POST /api/admin/settings/integrations/{channel}/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostIntegrationTest)),
	)
	// admin / error reporting (Sentry or GlitchTip)
	s.handle("GET /api/admin/settings/error-reporting", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetErrorReporting)),
	)
	s.handle("PUT /api/admin/settings/error-reporting", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutErrorReporting)),
	)
	s.handle("POST /api/admin/settings/error-reporting/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostErrorReportingTest)),
	)
	// admin / backups
	s.handle("GET /api/admin/backups/status", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupStatus)),
	)
	s.handle("GET /api/admin/backups/export", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupExport)),
	)
	s.handle("POST /api/admin/backups/import", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupImport)),
	)
	s.handle("POST /api/admin/backups/apply", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupApply)),
	)
	s.handle("POST /api/admin/backups/s3/run", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminBackupS3Run)),
	)
	// admin / scheduled S3 backup settings
	s.handle("GET /api/admin/settings/backup-s3", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminGetBackupS3Settings)),
	)
	s.handle("PUT /api/admin/settings/backup-s3", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPutBackupS3Settings)),
	)
	s.handle("POST /api/admin/settings/backup-s3/test", protected(
		s.requireScope(auth.ScopeAdminTools, s.handleAdminPostBackupS3Test)),
	)
}

func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.handler)
}
