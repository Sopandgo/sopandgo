package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
)

// Scheduled S3 backups: settings live in SQLite and apply without a restart.

func (s *Server) handleAdminGetBackupS3Settings(w http.ResponseWriter, r *http.Request) {
	out, err := s.s3Settings.GetPublic()
	if err != nil {
		http.Error(w, "failed to load S3 backup settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAdminPutBackupS3Settings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled         bool   `json:"enabled"`
		Bucket          string `json:"bucket"`
		Region          string `json:"region"`
		KeyPrefix       string `json:"key_prefix"`
		Endpoint        string `json:"endpoint"`
		UsePathStyle    bool   `json:"use_path_style"`
		AccessKeyID     string `json:"access_key_id"`
		SecretAccessKey string `json:"secret_access_key"`
		Interval        string `json:"interval"`
		RetentionMax    int    `json:"retention_max"`
		RetentionDays   int    `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err := s.s3Settings.Save(backup.SaveS3Input{
		Enabled:         req.Enabled,
		Bucket:          req.Bucket,
		Region:          req.Region,
		KeyPrefix:       req.KeyPrefix,
		Endpoint:        req.Endpoint,
		UsePathStyle:    req.UsePathStyle,
		AccessKeyID:     req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		Interval:        req.Interval,
		RetentionMax:    req.RetentionMax,
		RetentionDays:   req.RetentionDays,
	})
	switch {
	case errors.Is(err, backup.ErrS3KeyMissing):
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	case errors.Is(err, backup.ErrInvalidS3Settings):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case err != nil:
		log.Printf("save s3 backup settings: %v", err)
		http.Error(w, "failed to save S3 backup settings", http.StatusInternalServerError)
		return
	}

	saved, _ := s.s3Settings.GetPublic()
	actorID := GetUserID(r.Context())
	if saved != nil {
		s.writeAudit(audit.EventBackupS3SettingsUpdated, audit.EntitySystem, "backup", &actorID, map[string]any{
			"enabled":            saved.Enabled,
			"bucket":             saved.Bucket,
			"region":             saved.Region,
			"key_prefix":         saved.KeyPrefix,
			"endpoint":           saved.Endpoint,
			"interval":           saved.Interval,
			"retention_max":      saved.RetentionMax,
			"retention_days":     saved.RetentionDays,
			"static_credentials": saved.AccessKeyID != "",
		})
	}

	if err := s.applyBackupS3Settings(r.Context()); err != nil {
		log.Printf("apply s3 backup settings: %v", err)
		http.Error(w, "settings saved, but the scheduler could not use them: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyBackupS3Settings hands the saved settings to the running scheduler.
func (s *Server) applyBackupS3Settings(ctx context.Context) error {
	cfg, err := s.s3Settings.LoadConfig()
	if err != nil {
		return err
	}
	return s.s3Backup.Configure(ctx, cfg)
}

func (s *Server) handleAdminPostBackupS3Test(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.s3Settings.LoadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := backup.CheckS3Connection(ctx, cfg); err != nil {
		if errors.Is(err, backup.ErrS3NotConfigured) {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}
		http.Error(w, "S3 connection test failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleAdminBackupS3Run(w http.ResponseWriter, r *http.Request) {
	key, err := s.s3Backup.RunNow(r.Context())
	switch {
	case errors.Is(err, backup.ErrS3NotEnabled):
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
		return
	case errors.Is(err, backup.ErrBusy):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "S3 backup failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"object_key": key})
}
