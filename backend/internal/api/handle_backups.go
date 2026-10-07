package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
)

func (s *Server) handleAdminBackupStatus(w http.ResponseWriter, r *http.Request) {
	locked, message := s.backupSvc.IsLocked()
	pendingRestore := s.backupSvc.HasPendingRestore()
	w.Header().Set("Content-Type", "application/json")
	status := map[string]any{
		"locked":          locked,
		"message":         message,
		"pending_restore": pendingRestore,
	}
	if s.s3Backup != nil {
		st := s.s3Backup.Status()
		status["s3_scheduled"] = st
	} else {
		status["s3_scheduled"] = backup.S3SchedulerStatus{Enabled: false}
	}
	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) handleAdminBackupExport(w http.ResponseWriter, r *http.Request) {
	zipPath, fileName, manifest, err := s.backupSvc.ExportZip()
	if err != nil {
		if errors.Is(err, backup.ErrBusy) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "failed to export backup", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(filepath.Dir(zipPath))

	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventBackupExported, audit.EntitySystem, "backup", &actorID, map[string]any{
		"manifest": manifest,
	})

	f, err := os.Open(zipPath)
	if err != nil {
		http.Error(w, "failed to read backup archive", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	_, _ = io.Copy(w, f)
}

func (s *Server) handleAdminBackupImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "missing backup file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := s.backupSvc.ValidateImportArchive(file)
	if err != nil {
		switch {
		case errors.Is(err, backup.ErrBusy):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, backup.ErrPendingRestore):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, backup.ErrInvalidBackupArchive):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case errors.Is(err, backup.ErrUnsupportedBackupFormat),
			errors.Is(err, backup.ErrSchemaTooNew):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		default:
			http.Error(w, "failed to validate backup archive", http.StatusInternalServerError)
			return
		}
	}

	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventBackupImportValidated, audit.EntitySystem, "backup", &actorID, map[string]any{
		"manifest": result.Manifest,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"manifest": result.Manifest,
		"note":     "Import validation passed. Applying destructive restore is a separate operation.",
	})
}

func (s *Server) handleAdminBackupApply(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	confirmation := r.FormValue("confirmation")

	file, _, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "missing backup file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := s.backupSvc.StageImportApply(file, confirmation)
	if err != nil {
		switch {
		case errors.Is(err, backup.ErrBusy):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, backup.ErrPendingRestore):
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case errors.Is(err, backup.ErrApplyConfirmation):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case errors.Is(err, backup.ErrInvalidBackupArchive):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		case errors.Is(err, backup.ErrUnsupportedBackupFormat),
			errors.Is(err, backup.ErrSchemaTooNew):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		default:
			http.Error(w, "failed to stage backup apply", http.StatusInternalServerError)
			return
		}
	}

	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventBackupApplyStaged, audit.EntitySystem, "backup", &actorID, map[string]any{
		"manifest":         result.Manifest,
		"requires_restart": result.RequiresRestart,
		"pre_apply_backup": result.PreApplyBackup,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":               true,
		"manifest":         result.Manifest,
		"requires_restart": result.RequiresRestart,
		"pre_apply_backup": result.PreApplyBackup,
		"note":             "Backup restore is staged. Restart the backend/container to apply it.",
	})
}
