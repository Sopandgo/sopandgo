package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

func (s *Server) handleAdminListAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 50
	if l := query.Get("limit"); l != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(l))
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(o))
		if err != nil || parsed < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	filters := audit.ListFilters{
		EventType:   strings.ToLower(strings.TrimSpace(query.Get("type"))),
		EntityType:  strings.ToLower(strings.TrimSpace(query.Get("entity_type"))),
		ActorUserID: query.Get("actor_user_id"),
		SOPID:       query.Get("sop_id"),
	}

	// 1. Capture the 'total' variable here
	events, total, err := s.auditLogger.List(limit, offset, filters)
	if err != nil {
		http.Error(w, "failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	// 2. Wrap the result in a struct to send both list and count
	response := struct {
		Events []audit.AuditEventWithVerification `json:"events"`
		Total  int                                `json:"total"`
	}{
		Events: events,
		Total:  total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAdminAuditFilterOptions(w http.ResponseWriter, r *http.Request) {
	options, err := s.auditLogger.ListFilterOptions()
	if err != nil {
		http.Error(w, "failed to fetch audit filter options", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}
