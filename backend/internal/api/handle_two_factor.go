package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

// writeTwoFactorError maps service errors to stable JSON error codes the UI
// can translate. It returns false when err is nil.
func (s *Server) writeTwoFactorError(w http.ResponseWriter, r *http.Request, userID string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, auth.ErrWrongCurrentPassword):
		s.writeAudit(audit.EventLoginFailed, audit.EntityUser, userID, &userID, map[string]string{
			"failure_reason": "invalid_current_password",
			"ip_address":     r.RemoteAddr,
			"user_agent":     r.UserAgent(),
		})
		writeJSONError(w, http.StatusForbidden, "wrong_password", err.Error())
	case errors.Is(err, auth.ErrNoEncryptionKey):
		writeJSONError(w, http.StatusConflict, "encryption_key_missing", err.Error())
	case errors.Is(err, auth.ErrTwoFactorUnavailable):
		writeJSONError(w, http.StatusConflict, "authenticator_unavailable", err.Error())
	case errors.Is(err, auth.ErrInvalidSecondFactor):
		writeJSONError(w, http.StatusBadRequest, "invalid_code", err.Error())
	case errors.Is(err, auth.ErrNoPendingTwoFactor):
		writeJSONError(w, http.StatusConflict, "no_pending_setup", err.Error())
	case errors.Is(err, auth.ErrTwoFactorNotEnabled):
		writeJSONError(w, http.StatusConflict, "not_enabled", err.Error())
	case errors.Is(err, auth.ErrTwoFactorRequired):
		writeJSONError(w, http.StatusConflict, "required_by_org", err.Error())
	case errors.Is(err, auth.ErrRecoveryCodesAdminOnly):
		writeJSONError(w, http.StatusForbidden, "admin_only", err.Error())
	case errors.Is(err, auth.ErrCannotResetAdmin):
		writeJSONError(w, http.StatusForbidden, "admin_target", err.Error())
	case errors.Is(err, auth.ErrActorNeedsTwoFactor):
		writeJSONError(w, http.StatusConflict, "actor_needs_two_factor", err.Error())
	default:
		log.Printf("ERROR: two-factor request for user %s failed: %v", userID, err)
		http.Error(w, "two-factor request failed", http.StatusInternalServerError)
	}
	return true
}

func (s *Server) handleGetMyTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	status, err := s.authService.TwoFactorStatus(userID)
	if s.writeTwoFactorError(w, r, userID, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleBeginTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	var req struct {
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	setup, err := s.authService.BeginTOTPSetup(userID, req.CurrentPassword)
	if s.writeTwoFactorError(w, r, userID, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(setup)
}

func (s *Server) handleConfirmTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	var req struct {
		Code string `json:"code"`
		// RefreshToken is the caller's own session; every other session is signed out.
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	codes, err := s.authService.ConfirmTOTPSetup(userID, req.Code, req.RefreshToken)
	if s.writeTwoFactorError(w, r, userID, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"recovery_codes": codes})
}

func (s *Server) handleDisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	var req struct {
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	err := s.authService.DisableTwoFactor(userID, req.CurrentPassword)
	if s.writeTwoFactorError(w, r, userID, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	var req struct {
		CurrentPassword string `json:"current_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	codes, err := s.authService.RegenerateRecoveryCodes(userID, req.CurrentPassword)
	if s.writeTwoFactorError(w, r, userID, err) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{"recovery_codes": codes})
}

// handleAdminResetUserTwoFactor clears a normal user's second factor so they
// enroll again. Admin accounts are refused (403 admin_target).
func (s *Server) handleAdminResetUserTwoFactor(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("userID")
	if targetID == "" {
		http.Error(w, "missing userID", http.StatusBadRequest)
		return
	}
	actorID := GetUserID(r.Context())
	err := s.authService.ResetTwoFactor(targetID, &actorID)
	if s.writeTwoFactorError(w, r, actorID, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type securitySettingsResponse struct {
	RequireTwoFactor        bool `json:"require_two_factor"`
	EncryptionKeyConfigured bool `json:"encryption_key_configured"`
	EmailFallbackAvailable  bool `json:"email_fallback_available"`
}

func (s *Server) securitySettings() (*securitySettingsResponse, error) {
	required, err := s.authService.RequireTwoFactor()
	if err != nil {
		return nil, err
	}
	return &securitySettingsResponse{
		RequireTwoFactor:        required,
		EncryptionKeyConfigured: s.authService.EncryptionKeyConfigured(),
		EmailFallbackAvailable:  s.emailFallbackAvailable(),
	}, nil
}

func (s *Server) handleAdminGetSecuritySettings(w http.ResponseWriter, r *http.Request) {
	out, err := s.securitySettings()
	if err != nil {
		http.Error(w, "failed to read security settings", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAdminPatchSecuritySettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequireTwoFactor *bool `json:"require_two_factor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RequireTwoFactor == nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	actorID := GetUserID(r.Context())
	err := s.authService.SetRequireTwoFactor(*req.RequireTwoFactor, actorID)
	if s.writeTwoFactorError(w, r, actorID, err) {
		return
	}
	s.handleAdminGetSecuritySettings(w, r)
}
