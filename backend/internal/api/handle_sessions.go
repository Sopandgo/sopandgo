package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleAdminListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.authService.ListActiveSessions()
	if err != nil {
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleAdminRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		http.Error(w, "missing user ID", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	if err := s.authService.RevokeAllUserSessions(userID, &actorID); err != nil {
		http.Error(w, "failed to revoke sessions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAdminRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	// Decode and check for an empty reason
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		http.Error(w, "a reason must be provided", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	if err := s.authService.RevokeAllSessions(req.Reason, &actorID); err != nil {
		http.Error(w, "failed to revoke sessions", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
