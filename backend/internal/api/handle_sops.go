package api

import (
	"encoding/json"
	"fmt"
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

	// Pass all parameters to the service
	sops, total, err := s.sopService.ListSOPs(actorID, limit, offset, tagID, searchQuery, favoritesOnly, favoritesFirst)
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
