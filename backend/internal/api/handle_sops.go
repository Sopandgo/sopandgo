package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/sop"
)

func parseQueryBool(q url.Values, key string) bool {
	vals, ok := q[key]
	if !ok || len(vals) == 0 {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(vals[0]))
	return v == "true" || v == "1" || v == "yes"
}

type idResponse struct {
	ID string `json:"id"`
}

func (s *Server) handleRegisterSOP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	sopID, err := s.sopService.RegisterSOP(req.Title, &actorID)
	if err != nil {
		http.Error(w, "Failed to register SOP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(idResponse{ID: sopID})
}

func (s *Server) handleListSOPs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 50
	if l := query.Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	tagID := query.Get("tag_id")
	searchQuery := query.Get("q") // NEW: Extract the search query

	actorID := GetUserID(r.Context())
	favoritesOnly := parseQueryBool(query, "favorites_only")
	favoritesFirst := parseQueryBool(query, "favorites_first")

	// archived=only lists archived SOPs, archived=all both; anything else, SOPs in use
	archived := sop.ArchiveFilterActive
	switch strings.ToLower(strings.TrimSpace(query.Get("archived"))) {
	case "only":
		archived = sop.ArchiveFilterOnly
	case "all":
		archived = sop.ArchiveFilterAll
	}

	// Pass all parameters to the service
	sops, total, err := s.sopService.ListSOPs(actorID, limit, offset, tagID, searchQuery, favoritesOnly, favoritesFirst, archived)
	if err != nil {
		http.Error(w, "Failed to retrieve sops", http.StatusInternalServerError)
		return
	}

	response := struct {
		SOPs  []sop.SOPListItem `json:"sops"`
		Total int               `json:"total"`
	}{
		SOPs:  sops,
		Total: total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetSOPByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sopID")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())
	result, err := s.sopService.GetSOPByIDWithTags(id, actorID)
	if err != nil {
		http.Error(w, "SOP not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleFavoriteSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sop id", http.StatusBadRequest)
		return
	}
	actorID := GetUserID(r.Context())
	if err := s.sopService.FavoriteSOP(sopID, actorID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "SOP not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to favorite sop", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUnfavoriteSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sop id", http.StatusBadRequest)
		return
	}
	actorID := GetUserID(r.Context())
	if err := s.sopService.UnfavoriteSOP(sopID, actorID); err != nil {
		http.Error(w, "failed to unfavorite sop", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeSOPArchived answers a change to an archived SOP.
func writeSOPArchived(w http.ResponseWriter) {
	writeJSONError(w, http.StatusConflict, "sop_archived", "This SOP is archived. Unarchive it to make changes.")
}

func (s *Server) handleArchiveSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sop id", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}

	actorID := GetUserID(r.Context())
	err := s.sopService.ArchiveSOP(sopID, actorID, req.Reason)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, sop.ErrArchiveReasonRequired), errors.Is(err, sop.ErrArchiveReasonTooLong):
		writeJSONError(w, http.StatusBadRequest, "invalid_archive_reason", err.Error())
	case strings.HasPrefix(err.Error(), "conflict:"):
		writeJSONError(w, http.StatusConflict, "conflict", strings.TrimSpace(strings.TrimPrefix(err.Error(), "conflict:")))
	case strings.Contains(err.Error(), "not found"):
		writeJSONError(w, http.StatusNotFound, "sop_not_found", "SOP not found")
	default:
		log.Printf("ERROR: archive SOP %s failed: %v", sopID, err)
		http.Error(w, "failed to archive sop", http.StatusInternalServerError)
	}
}

func (s *Server) handleUnarchiveSOP(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sop id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())
	err := s.sopService.UnarchiveSOP(sopID, actorID)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case strings.HasPrefix(err.Error(), "conflict:"):
		writeJSONError(w, http.StatusConflict, "conflict", strings.TrimSpace(strings.TrimPrefix(err.Error(), "conflict:")))
	case strings.Contains(err.Error(), "not found"):
		writeJSONError(w, http.StatusNotFound, "sop_not_found", "SOP not found")
	default:
		log.Printf("ERROR: unarchive SOP %s failed: %v", sopID, err)
		http.Error(w, "failed to unarchive sop", http.StatusInternalServerError)
	}
}
