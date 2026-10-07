package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
)

func (s *Server) handleAdminGetSMTPSettings(w http.ResponseWriter, r *http.Request) {
	out, err := s.smtpSettings.GetPublic()
	if err != nil {
		http.Error(w, "failed to load email settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

type putSMTPSettingsRequest struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	FromAddress string `json:"from_address"`
	Password    string `json:"password"`
}

func (s *Server) handleAdminPutSMTPSettings(w http.ResponseWriter, r *http.Request) {
	var req putSMTPSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Host = strings.TrimSpace(req.Host)
	req.Port = strings.TrimSpace(req.Port)
	req.Username = strings.TrimSpace(req.Username)
	req.FromAddress = strings.TrimSpace(req.FromAddress)

	if req.Host == "" || req.Port == "" || req.Username == "" || req.FromAddress == "" {
		http.Error(w, "host, port, username, and from_address are required", http.StatusBadRequest)
		return
	}

	err := s.smtpSettings.Save(mail.SaveSMTPInput{
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		FromAddress: req.FromAddress,
		Password:    req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, mail.ErrSMTPKeyMissing):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		case errors.Is(err, mail.ErrSMTPPasswordRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, "failed to save smtp settings", http.StatusInternalServerError)
			return
		}
	}

	actorID := GetUserID(r.Context())
	payload := map[string]string{
		"host": req.Host,
		"port": req.Port,
		"from": req.FromAddress,
	}
	s.writeAudit(audit.EventSmtpSettingsUpdated, audit.EntitySystem, "smtp", &actorID, payload)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminPatchMailMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MailMode string `json:"mail_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.MailMode = strings.TrimSpace(req.MailMode)
	if req.MailMode == "" {
		http.Error(w, "mail_mode is required", http.StatusBadRequest)
		return
	}

	if err := s.smtpSettings.SetMailMode(req.MailMode); err != nil {
		if errors.Is(err, mail.ErrInvalidMailMode) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to update mail mode", http.StatusInternalServerError)
		return
	}

	actorID := GetUserID(r.Context())
	payload := map[string]string{
		"mail_mode": req.MailMode,
	}
	s.writeAudit(audit.EventMailModeUpdated, audit.EntitySystem, "mail", &actorID, payload)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminPatchDefaultLocale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DefaultLocale string `json:"default_locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := s.smtpSettings.SetDefaultLocale(req.DefaultLocale); err != nil {
		if errors.Is(err, i18n.ErrUnsupportedLocale) {
			http.Error(w, "unsupported locale", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to update default locale", http.StatusInternalServerError)
		return
	}
	tag, _ := i18n.Normalize(req.DefaultLocale)
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventDefaultLocaleUpdated, audit.EntitySystem, "locale", &actorID, map[string]string{
		"default_locale": tag,
	})
	w.WriteHeader(http.StatusNoContent)
}

type postSMTPTestRequest struct {
	To string `json:"to"`
}

func (s *Server) handleAdminPostSMTPTest(w http.ResponseWriter, r *http.Request) {
	var req postSMTPTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.To = strings.TrimSpace(req.To)
	if req.To == "" || !strings.Contains(req.To, "@") {
		http.Error(w, "valid \"to\" email is required", http.StatusBadRequest)
		return
	}

	err := s.smtpSettings.SendTestEmail(req.To)
	if err != nil {
		switch {
		case errors.Is(err, mail.ErrSMTPKeyMissing):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		case errors.Is(err, mail.ErrSMTPNotConfigured):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		case errors.Is(err, mail.ErrResendNotConfigured):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		default:
			http.Error(w, "failed to send test email: "+err.Error(), http.StatusBadGateway)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
