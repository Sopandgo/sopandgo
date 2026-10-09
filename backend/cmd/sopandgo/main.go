package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/api"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/pdfgen"
	"github.com/sopandgo/sopandgo/backend/internal/secrets"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
	_ "modernc.org/sqlite" // Pure Go SQLite driver (CGO-free)
)

func main() {
	log.Println("starting sopandgo backend")

	// 1. Setup Graceful Shutdown
	// We create a context that listens for OS interrupt signals (Ctrl+C, SIGINT).
	// This allows us to cleanly stop background tasks and close connections.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 2. Initialize Infrastructure (Database & Storage)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	log.Printf("using data directory: %s", dataDir)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("cannot create or access data directory: %v", err)
	}
	if err := backup.ApplyPendingRestoreAtStartup(dataDir); err != nil {
		log.Fatalf("failed to apply pending backup restore: %v", err)
	}

	// Storage manages the raw SQLite connection and file system paths.
	store, err := storage.Open(dataDir)
	if err != nil {
		log.Fatalf("cannot initialize storage: %v", err)
	}
	defer store.DB.Close()
	log.Println("database ready")
	// After a restore, put back this instance's S3 backup settings (the archive's are not used).
	if err := backup.ApplyCarriedS3Settings(store.DB, dataDir); err != nil {
		log.Fatalf("failed to keep S3 backup settings across restore: %v", err)
	}

	// 3. Initialize Core Services (Dependency Injection)

	// Audit Logger: Centralized logging for compliance events.
	auditLogger, err := audit.New(store.DB)
	if err != nil {
		log.Fatalf("Could not initialize audit logger: %v", err)
	}

	// Log the startup event for compliance tracking.
	if err := auditLogger.Log(
		store.DB,
		audit.EventSystemRestart,
		audit.EntitySystem,
		audit.EntitySystem,
		nil,
		`{"status": "ready"}`,
	); err != nil {
		log.Printf("audit write failed for system startup: %v", err)
	}

	// Auth Service: Manages users, roles, and sessions.
	authService := auth.NewService(store.DB, auditLogger)

	// Bootstrap: Ensure at least one admin exists so the system isn't locked out.
	if err := authService.EnsureAdminUser(); err != nil {
		log.Fatalf("failed to ensure admin user: %v", err)
	}
	adminUser, _, err := authService.GetUserByEmail("admin")
	if err != nil {
		log.Fatalf("failed to resolve admin user for startup tasks: %v", err)
	}

	// Background Task: Clean up expired sessions periodically.
	// We pass 'ctx' so this task stops automatically when the server shuts down.
	authService.StartSessionsCleanupTask(ctx, 24*time.Hour)

	// SOP Service: Manages Standard Operating Procedures and file versioning.
	pdfGeneratorVersion := os.Getenv("PDF_GENERATOR_VERSION")
	if pdfGeneratorVersion == "" {
		pdfGeneratorVersion = "1"
	}
	pdfExportEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("PDF_EXPORT_ENABLED"))) != "false"
	pdfRenderer := strings.ToLower(strings.TrimSpace(os.Getenv("PDF_RENDERER")))
	if pdfRenderer == "" {
		pdfRenderer = "gotenberg"
	}
	gotenbergURL := os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		gotenbergURL = "http://gotenberg:3000"
	}

	var renderer pdfgen.Renderer
	if pdfExportEnabled {
		switch pdfRenderer {
		case "gotenberg":
			if err := pdfgen.ValidateBaseURL(gotenbergURL); err != nil {
				log.Printf("WARNING: invalid GOTENBERG_URL (%v), disabling PDF export", err)
				pdfExportEnabled = false
			} else {
				renderer = pdfgen.NewGotenbergRenderer(gotenbergURL, 60*time.Second)
			}
		case "none":
			pdfExportEnabled = false
		default:
			log.Printf("WARNING: unknown PDF_RENDERER=%q, disabling PDF export", pdfRenderer)
			pdfExportEnabled = false
		}
	}

	sopService := sop.NewService(store.DB, auditLogger, dataDir, pdfGeneratorVersion, pdfExportEnabled, renderer)
	if sopService.IsPDFExportEnabled() {
		go func() {
			processed, generated, err := sopService.BackfillPDFArtifactsForGenerator(adminUser.ID)
			if err != nil {
				log.Printf("pdf artifact backfill failed: %v", err)
				return
			}
			log.Printf("pdf artifact backfill completed: processed=%d generated=%d generator_version=%s", processed, generated, pdfGeneratorVersion)
		}()
	} else {
		log.Println("PDF export is disabled")
	}

	// Mail and integration secrets live in SQLite (admin UI), encrypted with SECRET_ENCRYPTION_KEY.
	encKey, ok := secrets.KeyFromEnv()
	switch {
	case !ok && secrets.KeySet():
		log.Println("WARNING: SECRET_ENCRYPTION_KEY does not decode to 32 bytes (base64, hex, or 32 raw bytes); stored secrets cannot be saved or read")
	case !ok:
		log.Println("WARNING: SECRET_ENCRYPTION_KEY not set; set a 32-byte key (e.g. openssl rand -base64 32) to save SMTP and integration secrets and to send mail")
	}

	smtpSettingsStore := mail.NewSMTPSettingsStore(store.DB, encKey)
	mailSender := mail.NewDatabaseSender(smtpSettingsStore)
	integrationSettings := notify.NewSettingsStore(store.DB, encKey)
	notifyService := notify.NewService(integrationSettings, auditLogger)
	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "dev"
	}
	backupService := backup.NewService(store.DB, dataDir, appVersion, storage.LatestSchemaVersion())

	// Scheduled S3 backups: configured under Settings → Backup, applied without a restart.
	s3Settings := backup.NewS3SettingsStore(store.DB, encKey)
	s3Scheduler := backup.NewS3Scheduler(backupService, auditLogger)
	s3Scheduler.SetOnFailure(func(c context.Context, failErr error) {
		notifyService.Dispatch(c, notify.Event{
			Type:    notify.EventBackupS3Failed,
			Title:   i18n.T(i18n.ReadDefaultLocale(store.DB), "notify.backup_failed.title", nil),
			Message: failErr.Error(),
		})
	})
	if s3Cfg, err := s3Settings.LoadConfig(); err != nil {
		log.Printf("WARNING: S3 backups stay off until the settings are saved again: %v", err)
	} else if err := s3Scheduler.Configure(ctx, s3Cfg); err != nil {
		log.Printf("WARNING: S3 backups stay off until the settings are saved again: %v", err)
	} else if s3Cfg.Enabled {
		log.Printf("S3 automatic backups enabled: bucket=%s key_prefix=%s interval=%s",
			s3Cfg.Bucket, s3Scheduler.Status().KeyPrefix, s3Cfg.Interval)
	}

	mailService, err := mail.NewService(mailSender, auditLogger)
	if err != nil {
		log.Fatalf("Could not initialize mail service: %v", err)
	}

	// 4. Start API Server
	// The Server struct is "dumb"—it just routes HTTP requests to the services above.
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		// Fallback for local dev without docker
		origin = "http://localhost:8087"
		log.Println("WARNING: ORIGIN env var not set, defaulting to", origin)
	}

	// 2. Create Config Struct
	config := api.Config{
		Origin: origin,
	}

	apiServer := api.New(config, auditLogger, authService, sopService, mailService, smtpSettingsStore, notifyService, integrationSettings, backupService, s3Settings, s3Scheduler)
	log.Println("backend initialized successfully")

	s3Scheduler.Start(ctx)

	// Run the HTTP server in a separate goroutine so it doesn't block the main thread.
	// This allows the main thread to listen for the shutdown signal below.
	go func() {
		log.Println("HTTP server listening on :8080")
		if err := apiServer.Serve(":8080"); err != nil {
			log.Printf("HTTP server stopped: %v", err)
			// If the server crashes (e.g., port in use), cancel the context to exit the app.
			cancel()
		}
	}()

	// 5. Wait for Shutdown Signal
	// The main thread pauses here until Ctrl+C is pressed (or server errors out).
	<-ctx.Done()
	log.Println("Shutting down backend...")

	// Optional: Give background tasks a moment to finish current work.
	time.Sleep(500 * time.Millisecond)
	log.Println("Goodbye.")
}
