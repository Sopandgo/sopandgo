package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
)

func (s *Server) handleAdminGetIntegrations(w http.ResponseWriter, r *http.Request) {
	if s.integrationSettings == nil {
		http.Error(w, "integrations unavailable", http.StatusInternalServerError)
		return
	}
	out, err := s.integrationSettings.GetPublic()
	if err != nil {
		http.Error(w, "failed to load integration settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAdminPutSlackIntegration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled    bool     `json:"enabled"`
		WebhookURL string   `json:"webhook_url"`
		Events     []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err := s.integrationSettings.SaveSlack(notify.SaveSlackInput{
		Enabled:    req.Enabled,
		WebhookURL: strings.TrimSpace(req.WebhookURL),
		Events:     req.Events,
	})
	if err != nil {
		writeIntegrationSaveError(w, err)
		return
	}
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventIntegrationSettingsUpdated, audit.EntityIntegration, notify.ChannelSlack, &actorID, map[string]any{
		"channel": notify.ChannelSlack,
		"enabled": req.Enabled,
		"events":  req.Events,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminPutGotifyIntegration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool     `json:"enabled"`
		URL     string   `json:"url"`
		Token   string   `json:"token"`
		Events  []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err := s.integrationSettings.SaveGotify(notify.SaveGotifyInput{
		Enabled: req.Enabled,
		URL:     strings.TrimSpace(req.URL),
		Token:   req.Token,
		Events:  req.Events,
	})
	if err != nil {
		writeIntegrationSaveError(w, err)
		return
	}
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventIntegrationSettingsUpdated, audit.EntityIntegration, notify.ChannelGotify, &actorID, map[string]any{
		"channel": notify.ChannelGotify,
		"enabled": req.Enabled,
		"events":  req.Events,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminPutWebhookIntegration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled     bool     `json:"enabled"`
		URL         string   `json:"url"`
		BearerToken string   `json:"bearer_token"`
		Events      []string `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err := s.integrationSettings.SaveWebhook(notify.SaveWebhookInput{
		Enabled:     req.Enabled,
		URL:         strings.TrimSpace(req.URL),
		BearerToken: req.BearerToken,
		Events:      req.Events,
	})
	if err != nil {
		writeIntegrationSaveError(w, err)
		return
	}
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventIntegrationSettingsUpdated, audit.EntityIntegration, notify.ChannelWebhook, &actorID, map[string]any{
		"channel": notify.ChannelWebhook,
		"enabled": req.Enabled,
		"events":  req.Events,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminPostIntegrationTest(w http.ResponseWriter, r *http.Request) {
	channel := strings.TrimSpace(r.PathValue("channel"))
	switch channel {
	case notify.ChannelSlack, notify.ChannelGotify, notify.ChannelWebhook:
	default:
		http.Error(w, "unknown channel", http.StatusBadRequest)
		return
	}
	if s.notifyService == nil {
		http.Error(w, "integrations unavailable", http.StatusInternalServerError)
		return
	}
	err := s.notifyService.SendTest(r.Context(), channel, s.orgLocale())
	if err != nil {
		switch {
		case errors.Is(err, notify.ErrKeyMissing):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
		case errors.Is(err, notify.ErrNotConfigured),
			errors.Is(err, notify.ErrSlackWebhookRequired),
			errors.Is(err, notify.ErrGotifyTokenRequired),
			errors.Is(err, notify.ErrWebhookURLRequired):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
		default:
			http.Error(w, "failed to send test notification: "+err.Error(), http.StatusBadGateway)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func writeIntegrationSaveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, notify.ErrKeyMissing):
		http.Error(w, err.Error(), http.StatusPreconditionFailed)
	case errors.Is(err, notify.ErrInvalidEvent),
		errors.Is(err, notify.ErrSlackWebhookRequired),
		errors.Is(err, notify.ErrGotifyTokenRequired),
		errors.Is(err, notify.ErrWebhookURLRequired),
		errors.Is(err, notify.ErrNotConfigured):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		msg := err.Error()
		if strings.Contains(msg, "URL must be") || strings.Contains(msg, "https URL") {
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to save integration settings", http.StatusInternalServerError)
	}
}

// notifyLifecycle builds a SOP lifecycle event and fans it out (best-effort).
func (s *Server) notifyLifecycle(eventType, versionID, actorID, rejectReason string) {
	if s.notifyService == nil {
		return
	}
	version, err := s.sopService.GetSOPVersionByID(versionID)
	if err != nil {
		return
	}
	sopRow, err := s.sopService.GetSOPByID(version.SOPID)
	if err != nil {
		return
	}
	actorName := ""
	if actorID != "" {
		if u, err := s.authService.GetUserByID(actorID); err == nil && u != nil {
			actorName = u.DisplayName
		}
	}
	origin := strings.TrimRight(s.Config.Origin, "/")
	link := fmt.Sprintf("%s/sops/%s/v/%s", origin, version.SOPID, version.ID)

	locale := s.orgLocale()
	versionLabel := fmt.Sprintf("%d", version.Version)
	var title, message string
	switch eventType {
	case notify.EventSOPPublished:
		title = i18n.T(locale, "notify.sop_published.title", map[string]string{"title": sopRow.Title})
		message = i18n.T(locale, "notify.sop_published.message", map[string]string{"version": versionLabel})
		if version.ChangeSummary != "" {
			message += " " + version.ChangeSummary
		}
		link = fmt.Sprintf("%s/sops/%s/v/latest", origin, version.SOPID)
	case notify.EventSOPRC:
		title = i18n.T(locale, "notify.sop_rc.title", map[string]string{"title": sopRow.Title})
		message = i18n.T(locale, "notify.sop_rc.message", map[string]string{"version": versionLabel})
	case notify.EventSOPRejected:
		title = i18n.T(locale, "notify.sop_rejected.title", map[string]string{"title": sopRow.Title})
		message = i18n.T(locale, "notify.sop_rejected.message", map[string]string{"version": versionLabel})
		if rejectReason != "" {
			message += " " + i18n.T(locale, "notify.reason", map[string]string{"reason": rejectReason})
		}
	default:
		return
	}

	s.notifyService.Dispatch(context.Background(), notify.Event{
		Type:      eventType,
		Title:     title,
		Message:   message,
		URL:       link,
		SOPID:     version.SOPID,
		VersionID: version.ID,
		Version:   version.Version,
		ActorID:   actorID,
		ActorName: actorName,
	})
}
