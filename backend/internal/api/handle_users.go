package api

import (
	"encoding/json"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

func (s *Server) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	var req struct {
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate Password
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.authService.UpdateUserPassword(userID, req.NewPassword, &userID)
	if err != nil {
		http.Error(w, "failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Validate Password Complexity
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 2. Perform the Reset
	// - Hashes the token
	// - Checks the DB for validity/expiry
	// - Updates the password
	// - Burns the token (Single Use)
	if err := s.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		http.Error(w, "Password reset failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// 3. Success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully. You may now log in."})
}
