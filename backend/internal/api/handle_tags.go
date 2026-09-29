package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleCreateTag creates a new global tag.
// POST /api/tags
func (s *Server) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	id, err := s.sopService.CreateTag(req.Title, &actorID)
	if err != nil {
		// If the service layer threw a conflict error, return a 409
		if strings.Contains(err.Error(), "conflict") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "failed to create tag", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id": id,
	})
}

// handleListTags returns all available global tags.
// GET /api/tags
func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.sopService.ListTags()
	if err != nil {
		http.Error(w, "failed to list tags", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// handleSetTagStatus retires (soft-deletes) or revives a tag.
// PATCH /api/tags/{tagID}/status
func (s *Server) handleSetTagStatus(w http.ResponseWriter, r *http.Request) {
	tagID := r.PathValue("tagID")
	if tagID == "" {
		http.Error(w, "missing tag id", http.StatusBadRequest)
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	if err := s.sopService.SetTagStatus(tagID, req.IsActive, &actorID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "tag not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update tag status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleAttachTagToSOP links an existing tag to an SOP.
// POST /api/sops/{sopID}/tags/{tagID}
func (s *Server) handleAttachTagToSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	tagID := r.PathValue("tagID")

	if sopID == "" || tagID == "" {
		http.Error(w, "missing sop id or tag id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	if err := s.sopService.AttachTagToSOP(sopID, tagID, &actorID); err != nil {
		http.Error(w, "failed to attach tag to sop", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDetachTagFromSOP removes a tag link from an SOP.
// DELETE /api/sops/{sopID}/tags/{tagID}
func (s *Server) handleDetachTagFromSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	tagID := r.PathValue("tagID")

	if sopID == "" || tagID == "" {
		http.Error(w, "missing sop id or tag id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	if err := s.sopService.DetachTagFromSOP(sopID, tagID, &actorID); err != nil {
		http.Error(w, "failed to detach tag from sop", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content is standard for successful DELETE
}
