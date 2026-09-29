package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
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

		_ = s.auditLogger.Log(nil, audit.EventLoginFailed, audit.EntityUser, entityID, nil, payload)
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

	// 4. Generate PASETO
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		// System errors usually aren't logged as "security failures" but you could if you want
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// 5. Create Session (Service Call)
	refreshToken, err := s.authService.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// 6. Log Login
	_ = s.auditLogger.Log(nil, audit.EventLogin, audit.EntityUser, user.ID, &user.ID, map[string]string{
		"ip_address": r.RemoteAddr,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
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
