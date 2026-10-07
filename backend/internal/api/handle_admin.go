package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
)

type ExtendedIntegrityReport struct {
	OK        bool                  `json:"ok"`
	CheckedAt string                `json:"checked_at"`
	Audit     AuditIntegritySection `json:"audit"` // New detailed audit section
	Versions  sop.IntegritySection  `json:"versions"`
	Assets    sop.IntegritySection  `json:"assets"`
}

// Matches the shape of sop.IntegritySection, but uses audit-specific failure details
type AuditIntegritySection struct {
	Checked int                  `json:"checked"`
	Valid   int                  `json:"valid"`
	Invalid int                  `json:"invalid"`
	Failed  []audit.ChainFailure `json:"failed,omitempty"`
}

func (s *Server) handleAdminIntegrity(w http.ResponseWriter, r *http.Request) {
	// 1. Run checks
	sopReport, err := s.sopService.RunSystemIntegrityCheck()
	if err != nil {
		http.Error(w, "Failed to run SOP integrity check", http.StatusInternalServerError)
		return
	}

	auditReport, err := s.auditLogger.VerifyEntireChain()
	if err != nil {
		http.Error(w, "Failed to verify audit chain", http.StatusInternalServerError)
		return
	}

	// 2. Determine overall health (OK only if all sub-systems have 0 invalid entries)
	systemOK := sopReport.OK && (auditReport.Invalid == 0)

	// 3. Assemble the expanded DTO
	extendedReport := ExtendedIntegrityReport{
		OK:        systemOK,
		CheckedAt: sopReport.CheckedAt,
		Audit: AuditIntegritySection{
			Checked: auditReport.Checked,
			Valid:   auditReport.Valid,
			Invalid: auditReport.Invalid,
			Failed:  auditReport.Failed,
		},
		Versions: sopReport.Versions,
		Assets:   sopReport.Assets,
	}

	// 4. Audit Log the check
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventIntegrityCheck, audit.EntitySystem, audit.EntitySystem, &actorID, extendedReport)

	if !systemOK && s.notifyService != nil {
		locale := s.orgLocale()
		s.notifyService.Dispatch(r.Context(), notify.Event{
			Type:  notify.EventIntegrityCheckFailed,
			Title: i18n.T(locale, "notify.integrity.title", nil),
			Message: i18n.T(locale, "notify.integrity.message", map[string]string{
				"versions": fmt.Sprintf("%d", sopReport.Versions.Invalid),
				"assets":   fmt.Sprintf("%d", sopReport.Assets.Invalid),
				"audit":    fmt.Sprintf("%d", auditReport.Invalid),
			}),
			ActorID: actorID,
			Extra: map[string]any{
				"versions_invalid": sopReport.Versions.Invalid,
				"assets_invalid":   sopReport.Assets.Invalid,
				"audit_invalid":    auditReport.Invalid,
			},
		})
	}

	// 5. Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(extendedReport)
}

// handleAdminRegisterUser allows admins to create new accounts.
func (s *Server) handleAdminRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Role        string `json:"role"`
		Locale      string `json:"locale"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	locale := strings.TrimSpace(req.Locale)
	if locale == "" {
		locale = s.orgLocale()
	} else if _, ok := i18n.Normalize(locale); !ok {
		http.Error(w, "unsupported locale", http.StatusBadRequest)
		return
	}

	// 1. Register User
	// (Password is auto-generated as garbage internally, so we don't pass it here)
	userID, err := s.authService.RegisterUser(req.DisplayName, req.Email, req.Role, &actorID)
	if err != nil {
		http.Error(w, "failed to register user: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.authService.SetUserLocale(userID, locale); err != nil {
		http.Error(w, "failed to set user locale", http.StatusInternalServerError)
		return
	}

	// 2. Generate Invite Token
	// We need this token to create the "set-password" link.
	rawToken, _, err := s.authService.GeneratePasswordResetToken(userID)
	if err != nil {
		// Critical Logic: The user IS created, but we failed to generate the invite.
		// We log the error so the admin knows, but we return 200 OK because the DB write succeeded.
		// The Admin can just hit "Reset Password" later to fix this.
		log.Printf("ERROR: User %s created, but invite token generation failed: %v", userID, err)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":      userID,
			"warning": "User created, but invite token failed. Please trigger a manual password reset.",
		})
		return
	}

	// 3. Construct the URL
	inviteURL := fmt.Sprintf("%s/reset-password?token=%s", s.Config.Origin, rawToken)

	mailMode, modeErr := s.smtpSettings.GetMailMode()
	if modeErr != nil {
		log.Printf("ERROR: failed to read mail mode: %v", modeErr)
		mailMode = mail.MailModeSMTP
	}
	if mailMode == mail.MailModeManualLinks {
		s.writeAudit(audit.EventManualInviteLinkGenerated, audit.EntityUser, userID, &actorID, map[string]string{
			"delivery_mode": mail.MailModeManualLinks,
		})
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":            userID,
			"message":       "User created. Manual link mode is enabled.",
			"invite_link":   inviteURL,
			"delivery":      "manual_link",
			"delivery_hint": "Share this one-time link over a trusted channel.",
		})
		return
	}

	// 4. Send Welcome Email
	err = s.mailService.SendUserWelcomeEmail(req.Email, userID, req.DisplayName, inviteURL, locale)
	if err != nil {
		log.Printf("ERROR: User %s registered but Welcome Email failed: %v", userID, err)
		// We still return success because the user account exists.
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":      userID,
			"warning": "User created, but welcome email delivery failed. You can trigger a reset later or switch to manual links mode.",
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": userID})
}

func (s *Server) handleAdminUpdateRole(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		http.Error(w, "missing userID", http.StatusBadRequest)
		return
	}

	var req struct {
		NewRole string `json:"new_role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	err := s.authService.UpdateUserRole(userID, req.NewRole, &actorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminUpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		http.Error(w, "missing userID", http.StatusBadRequest)
		return
	}

	var req struct {
		Active bool `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	err := s.authService.SetUserActiveStatus(userID, req.Active, &actorID)
	if err != nil {
		if err == auth.ErrSelfDisable {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	// Simple Read (Store)
	users, err := s.authService.ListUsers()
	if err != nil {
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) handleAdminTriggerPasswordReset(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		http.Error(w, "missing userID", http.StatusBadRequest)
		return
	}

	// 1. Generate a fresh Token
	// We get the full user object back here, so we don't need to query for their email again.
	rawToken, user, err := s.authService.GeneratePasswordResetToken(userID)
	if err != nil {
		http.Error(w, "failed to generate reset token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Construct the URL
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.Config.Origin, rawToken)

	mailMode, modeErr := s.smtpSettings.GetMailMode()
	if modeErr != nil {
		log.Printf("ERROR: failed to read mail mode: %v", modeErr)
		mailMode = mail.MailModeSMTP
	}
	if mailMode == mail.MailModeManualLinks {
		actorID := GetUserID(r.Context())
		s.writeAudit(audit.EventManualResetLinkGenerated, audit.EntityUser, user.ID, &actorID, map[string]string{
			"delivery_mode": mail.MailModeManualLinks,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Reset link generated (manual link mode).",
			"link":    resetURL,
		})
		return
	}

	// 3. Send the Email
	// Using SendPasswordResetEmail (distinct from Welcome Email)
	err = s.mailService.SendPasswordResetEmail(user.Email, user.ID, user.DisplayName, resetURL, user.Locale)
	if err != nil {
		// Log the failure but return success to the admin UI (since the token is valid)
		log.Printf("ERROR: Admin triggered reset for %s, but email failed: %v", user.ID, err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Token generated, but email delivery failed. Check server logs or switch to manual links mode.",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password reset email sent."})
}
