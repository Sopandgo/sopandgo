package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/mail"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	// Use Store function
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleUpdateMyLocale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Locale string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	userID := GetUserID(r.Context())
	if err := s.authService.SetUserLocale(userID, req.Locale); err != nil {
		if errors.Is(err, auth.ErrInvalidLocale) {
			http.Error(w, "unsupported locale", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to update locale", http.StatusInternalServerError)
		return
	}
	tag, _ := i18n.Normalize(req.Locale)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"locale": tag})
}

func (s *Server) handleUpdateMyTheme(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Theme string `json:"theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	userID := GetUserID(r.Context())
	if err := s.authService.SetUserTheme(userID, req.Theme); err != nil {
		if errors.Is(err, auth.ErrInvalidTheme) {
			http.Error(w, "unsupported theme", http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to update theme", http.StatusInternalServerError)
		return
	}
	tag, _ := auth.NormalizeTheme(req.Theme)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"theme": tag})
}

func (s *Server) handleGetMySignatureStatus(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	statuses, err := s.sopService.GetSignatureStatusByUser(userID)
	if err != nil {
		http.Error(w, "Failed to get signature status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logFailure := func(reason string, userID *string) {
		// Log against the user if known, otherwise just log the attempt email
		payload := map[string]string{
			"attempted_email": req.Email,
			"failure_reason":  reason,
			"ip_address":      r.RemoteAddr,
			"user_agent":      r.UserAgent(),
		}

		entityID := req.Email
		if userID != nil {
			entityID = *userID
		}

		s.writeAudit(audit.EventLoginFailed, audit.EntityUser, entityID, nil, payload)
	}

	// 1. Fetch user
	user, hash, err := s.authService.GetUserByEmail(req.Email)
	if err != nil {
		// Log the failure before returning
		logFailure("user_not_found", nil)

		// Be vague about errors for security
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 2. Check Active
	if !user.IsActive {
		logFailure("account_inactive", &user.ID)
		http.Error(w, "Account is inactive", http.StatusForbidden)
		return
	}

	// 3. Verify password
	if err := auth.VerifyPassword(req.Password, hash); err != nil {
		logFailure("invalid_password", &user.ID)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 4. Second factor: the password alone does not sign in. Hand back a
	// short-lived challenge instead of a session.
	if user.TwoFactorEnabled {
		s.respondTwoFactorChallenge(w, user)
		return
	}

	s.completeLogin(w, r, user, "")
}

// respondTwoFactorChallenge answers a correct password for an account with 2FA.
func (s *Server) respondTwoFactorChallenge(w http.ResponseWriter, user *auth.User) {
	challenge, err := auth.GenerateTwoFactorChallenge(user.ID)
	if err != nil {
		http.Error(w, "Failed to start two-factor sign-in", http.StatusInternalServerError)
		return
	}
	methods, err := s.authService.SecondFactorMethods(user)
	if err != nil {
		http.Error(w, "Failed to start two-factor sign-in", http.StatusInternalServerError)
		return
	}
	if s.emailFallbackAvailable() {
		methods = append(methods, auth.SecondFactorEmail)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"two_factor_required": true,
		"challenge_token":     challenge,
		"methods":             methods,
		"expires_in":          int(auth.TwoFactorChallengeTTL.Seconds()),
	})
}

// completeLogin issues the access and refresh tokens and writes the login event.
func (s *Server) completeLogin(w http.ResponseWriter, r *http.Request, user *auth.User, secondFactor string) {
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := s.authService.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	payload := map[string]string{
		"ip_address": r.RemoteAddr,
	}
	if secondFactor != "" {
		payload["second_factor"] = secondFactor
	}
	s.writeAudit(audit.EventLogin, audit.EntityUser, user.ID, &user.ID, payload)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// emailFallbackAvailable reports whether sign-in codes can be mailed: only
// when a mail transport is saved and switched on.
func (s *Server) emailFallbackAvailable() bool {
	if s.smtpSettings == nil || s.mailService == nil {
		return false
	}
	mode, err := s.smtpSettings.EffectiveMailMode()
	return err == nil && mode == mail.MailModeSMTP
}

// challengeUser resolves the challenge token to an active account.
func (s *Server) challengeUser(w http.ResponseWriter, challengeToken string) (*auth.User, bool) {
	userID, err := auth.VerifyTwoFactorChallenge(challengeToken)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "challenge_expired", "The sign-in took too long. Enter your password again.")
		return nil, false
	}
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "challenge_expired", "The sign-in took too long. Enter your password again.")
		return nil, false
	}
	if !user.IsActive {
		writeJSONError(w, http.StatusForbidden, "account_inactive", "Account is inactive")
		return nil, false
	}
	return user, true
}

func (s *Server) handleLoginTwoFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChallengeToken string `json:"challenge_token"`
		Method         string `json:"method"`
		Code           string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user, ok := s.challengeUser(w, req.ChallengeToken)
	if !ok {
		return
	}
	// The IP limiter alone does not stop guesses spread over many addresses.
	if !s.twoFactorLimiter.getLimiter("user:" + user.ID).Allow() {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	method := req.Method
	if method == "" {
		method = auth.SecondFactorTOTP
	}
	if method == auth.SecondFactorEmail && !s.emailFallbackAvailable() {
		writeJSONError(w, http.StatusBadRequest, "email_unavailable", "Email codes are not available on this server.")
		return
	}

	remaining, err := s.authService.VerifySecondFactor(user.ID, method, req.Code)
	switch {
	case errors.Is(err, auth.ErrTwoFactorUnavailable):
		log.Printf("ERROR: cannot read the authenticator secret of user %s (SECRET_ENCRYPTION_KEY missing or changed?). "+
			"Reset normal users from Admin -> Users; reset an admin with `./sopandgo recover-admin --email <email>`: %v", user.ID, err)
		writeJSONError(w, http.StatusConflict, "authenticator_unavailable", "The authenticator for this account cannot be checked. Use another method or contact your admin.")
		return
	case errors.Is(err, auth.ErrUnknownSecondFactorType):
		writeJSONError(w, http.StatusBadRequest, "unknown_method", err.Error())
		return
	case err != nil:
		if !errors.Is(err, auth.ErrInvalidSecondFactor) && !errors.Is(err, auth.ErrTwoFactorNotEnabled) &&
			!errors.Is(err, auth.ErrRecoveryCodesAdminOnly) {
			log.Printf("ERROR: second factor check failed for user %s: %v", user.ID, err)
		}
		s.writeAudit(audit.EventLoginFailed, audit.EntityUser, user.ID, nil, map[string]string{
			"attempted_email": user.Email,
			"failure_reason":  "invalid_second_factor",
			"second_factor":   method,
			"ip_address":      r.RemoteAddr,
			"user_agent":      r.UserAgent(),
		})
		writeJSONError(w, http.StatusUnauthorized, "invalid_code", "Invalid verification code")
		return
	}

	if method == auth.SecondFactorRecoveryCode {
		w.Header().Set("X-Recovery-Codes-Remaining", fmt.Sprintf("%d", remaining))
	}
	s.completeLogin(w, r, user, method)
}

func (s *Server) handleLoginTwoFactorEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChallengeToken string `json:"challenge_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user, ok := s.challengeUser(w, req.ChallengeToken)
	if !ok {
		return
	}
	if !s.emailFallbackAvailable() {
		writeJSONError(w, http.StatusBadRequest, "email_unavailable", "Email codes are not available on this server.")
		return
	}

	code, err := s.authService.IssueEmailCode(user.ID)
	switch {
	case errors.Is(err, auth.ErrEmailCodeThrottled):
		w.Header().Set("Retry-After", "60")
		writeJSONError(w, http.StatusTooManyRequests, "email_throttled", err.Error())
		return
	case err != nil:
		http.Error(w, "Failed to create sign-in code", http.StatusInternalServerError)
		return
	}
	if err := s.mailService.SendTwoFactorCodeEmail(user.Email, user.ID, user.DisplayName, code, user.Locale); err != nil {
		log.Printf("ERROR: sign-in code email to user %s failed: %v", user.ID, err)
		writeJSONError(w, http.StatusBadGateway, "email_failed", "The code could not be sent.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Pass nil for actorID. Your RevokeSession method should be updated
	// to find the session by token string alone, if it doesn't already.
	err := s.authService.RevokeSession(req.RefreshToken, nil)

	if err != nil {
		// Log it, but don't tell the user "logout failed" — just let them go
		log.Printf("Logout failed: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	userID, role, err := s.authService.ValidateSession(req.RefreshToken)
	if err != nil {
		http.Error(w, "invalid or expired session", http.StatusUnauthorized)
		return
	}

	newAccessToken, err := auth.GenerateAccessToken(userID, role)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"access_token": newAccessToken,
	})
}
