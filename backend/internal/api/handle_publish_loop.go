package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
)

func (s *Server) handleDiffSOPVersion(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	versionID := r.PathValue("sopVersionID")
	if sopID == "" || versionID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	diff, err := s.sopService.DiffSOPVersion(sopID, versionID, r.URL.Query().Get("against"))
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "not part of this SOP") {
			http.Error(w, "Version not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to compare versions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

func (s *Server) handleListRecentPublishes(w http.ResponseWriter, r *http.Request) {
	limit := 12
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = n
	}

	items, err := s.sopService.ListRecentPublishes(limit)
	if err != nil {
		http.Error(w, "Failed to list recent publishes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleTrainingCoverage(w http.ResponseWriter, r *http.Request) {
	s.writeTrainingCoverage(w, "")
}

func (s *Server) handleSOPTrainingCoverage(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	s.writeTrainingCoverage(w, sopID)
}

func (s *Server) writeTrainingCoverage(w http.ResponseWriter, sopID string) {
	users, err := s.authService.ListUsers()
	if err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}
	rows, err := s.sopService.TrainingCoverage(sopID, users)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "SOP not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load signature coverage", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

// notifyVersionPublished emails active readers when mail can actually be sent.
// Manual-link mode and delivery failures leave the published version in place.
func (s *Server) notifyVersionPublished(versionID, actorID string) {
	mode, err := s.smtpSettings.GetMailMode()
	if err != nil {
		log.Printf("publish notice skipped: mail mode: %v", err)
		return
	}
	if mode == mail.MailModeManualLinks {
		return
	}

	version, err := s.sopService.GetSOPVersionByID(versionID)
	if err != nil {
		log.Printf("publish notice skipped: version %s: %v", versionID, err)
		return
	}
	sopRow, err := s.sopService.GetSOPByID(version.SOPID)
	if err != nil {
		log.Printf("publish notice skipped: sop %s: %v", version.SOPID, err)
		return
	}
	users, err := s.authService.ListUsers()
	if err != nil {
		log.Printf("publish notice skipped: users: %v", err)
		return
	}

	link := fmt.Sprintf("%s/sops/%s/v/latest", strings.TrimRight(s.Config.Origin, "/"), version.SOPID)
	for _, user := range users {
		if !user.IsActive || user.ID == actorID || strings.TrimSpace(user.Email) == "" {
			continue
		}
		if !auth.HasPermission(user.Role, auth.ScopeSOPSignReader) {
			continue
		}
		if err := s.mailService.SendSOPPublishedEmail(
			user.Email,
			user.DisplayName,
			sopRow.Title,
			version.Version,
			version.ChangeSummary,
			link,
		); err != nil {
			log.Printf("publish notice failed for %s: %v", user.Email, err)
		}
	}
}
