package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
)

func (s *Server) handleAddAcknowledgmentReader(w http.ResponseWriter, r *http.Request) {
	s.processHandleAddAcknowledgment(w, r, "reader")
}

func (s *Server) processHandleAddAcknowledgment(w http.ResponseWriter, r *http.Request, ackType string) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	sopVersionID := r.PathValue("sopVersionID")
	if sopVersionID == "" {
		http.Error(w, "missing sopVersionID", http.StatusBadRequest)
		return
	}

	userID := GetUserID(r.Context())

	id, err := s.sopService.AddAcknowledgment(
		sopVersionID,
		userID,
		ackType,
	)
	if err != nil {
		switch {
		case errors.Is(err, sop.ErrSOPArchived):
			writeSOPArchived(w)
		case errors.Is(err, errkind.Conflict):
			writeKindText(w, err, err.Error())
		case errors.Is(err, errkind.NotFound):
			writeKindText(w, err, "sop version not found")
		default:
			http.Error(w, "Failed to record acknowledgment", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *Server) handleListVersionAcknowledgments(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	sopVersionID := r.PathValue("sopVersionID")
	if sopVersionID == "" {
		http.Error(w, "missing sopVersionID", http.StatusBadRequest)
		return
	}

	acks, err := s.sopService.GetAcknowledgmentsByVersionWithUser(sopVersionID)
	if err != nil {
		http.Error(w, "Failed to retrieve acknowledgments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acks)
}
