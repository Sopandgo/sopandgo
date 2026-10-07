package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
)

func (s *Server) handleAdminPatchMailTransport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MailTransport string `json:"mail_transport"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.MailTransport = strings.TrimSpace(req.MailTransport)
	if req.MailTransport == "" {
		http.Error(w, "mail_transport is required", http.StatusBadRequest)
		return
	}

	if err := s.smtpSettings.SetMailTransport(req.MailTransport); err != nil {
		if errors.Is(err, mail.ErrInvalidMailTransport) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to update mail transport", http.StatusInternalServerError)
		return
	}

	actorID := GetUserID(r.Context())
	payload := map[string]string{
		"mail_transport": req.MailTransport,
	}
	s.writeAudit(audit.EventMailTransportUpdated, audit.EntitySystem, "mail_transport", &actorID, payload)

	w.WriteHeader(http.StatusNoContent)
}

type putResendSettingsRequest struct {
	FromAddress string `json:"from_address"`
	APIKey      string `json:"api_key"`
}

func (s *Server) handleAdminPutResendSettings(w http.ResponseWriter, r *http.Request) {
	var req putResendSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.FromAddress = strings.TrimSpace(req.FromAddress)
	if req.FromAddress == "" {
		http.Error(w, "from_address is required", http.StatusBadRequest)
		return
	}

	err := s.smtpSettings.SaveResend(mail.SaveResendInput{
		FromAddress: req.FromAddress,
		APIKey:      req.APIKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, mail.ErrSMTPKeyMissing):
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		case errors.Is(err, mail.ErrResendAPIKeyRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, "failed to save resend settings", http.StatusInternalServerError)
			return
		}
	}

	actorID := GetUserID(r.Context())
	payload := map[string]string{
		"from": req.FromAddress,
	}
	s.writeAudit(audit.EventResendSettingsUpdated, audit.EntitySystem, "resend", &actorID, payload)

	w.WriteHeader(http.StatusNoContent)
}
