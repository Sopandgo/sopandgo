package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/anchor"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// SetAuditAnchoring enables the audit anchoring endpoints and the anchoring
// part of the integrity scan. Without it those endpoints answer 503.
func (s *Server) SetAuditAnchoring(svc *anchor.Service) {
	s.anchoring = svc
}

func (s *Server) anchoringOr503(w http.ResponseWriter) bool {
	if s.anchoring == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "anchoring_unavailable", "audit anchoring is not available on this server")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeAnchoringError maps service errors to responses. TSA and S3 failures
// are the admin's to fix, so their message is returned.
func writeAnchoringError(w http.ResponseWriter, err error) {
	var nc *anchor.NeedsConfirmation
	switch {
	case errors.As(err, &nc):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "confirmation_required",
			"detail": nc.Error(),
			"probe":  nc.Probe,
		})
	case errors.Is(err, anchor.ErrInvalidSettings):
		writeJSONError(w, http.StatusBadRequest, "invalid_settings", err.Error())
	case errors.Is(err, anchor.ErrKeyMissing), errors.Is(err, anchor.ErrS3NotConfigured):
		writeJSONError(w, http.StatusPreconditionFailed, "precondition_failed", err.Error())
	case errors.Is(err, anchor.ErrNotEnabled):
		writeJSONError(w, http.StatusConflict, "not_enabled", err.Error())
	default:
		log.Printf("audit anchoring: %v", err)
		writeJSONError(w, http.StatusBadGateway, "anchoring_failed", err.Error())
	}
}

// GET /api/admin/audit-anchoring
func (s *Server) handleAdminGetAuditAnchoring(w http.ResponseWriter, r *http.Request) {
	if !s.anchoringOr503(w) {
		return
	}
	st, err := s.anchoring.Status()
	if err != nil {
		log.Printf("audit anchoring status: %v", err)
		http.Error(w, "failed to read audit anchoring status", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// POST /api/admin/audit-anchoring/probe sends one test request to the TSA and
// reports what would be pinned, without saving anything.
func (s *Server) handleAdminProbeAuditAnchoring(w http.ResponseWriter, r *http.Request) {
	if !s.anchoringOr503(w) {
		return
	}
	var in anchor.SettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	res, err := s.anchoring.Probe(r.Context(), in)
	if err != nil {
		writeAnchoringError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// PUT /api/admin/audit-anchoring switches anchoring on, off, or reconfigures
// it. Switching on, changing the TSA, or re-pinning answers 409 with the probe
// result until the request repeats it with confirm_fingerprint (and, when
// switching on, accept_broken).
func (s *Server) handleAdminPutAuditAnchoring(w http.ResponseWriter, r *http.Request) {
	if !s.anchoringOr503(w) {
		return
	}
	var in anchor.SettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	actorID := GetUserID(r.Context())
	if err := s.anchoring.Save(r.Context(), in, &actorID); err != nil {
		writeAnchoringError(w, err)
		return
	}
	st, _ := s.anchoring.Status()
	writeJSON(w, http.StatusOK, st)
}

// POST /api/admin/audit-anchoring/run takes an anchor now.
func (s *Server) handleAdminRunAuditAnchoring(w http.ResponseWriter, r *http.Request) {
	if !s.anchoringOr503(w) {
		return
	}
	a, err := s.anchoring.AnchorNow(r.Context())
	if err != nil && a == nil {
		writeAnchoringError(w, err)
		return
	}
	resp := map[string]any{"anchor": a, "skipped": a == nil}
	if err != nil {
		// Anchored, but the receipt is not in S3 yet.
		resp["warning"] = err.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/admin/audit-anchoring/bundle downloads the evidence bundle.
func (s *Server) handleAdminAuditEvidenceBundle(w http.ResponseWriter, r *http.Request) {
	if !s.anchoringOr503(w) {
		return
	}
	name := anchor.BundleFileName(time.Now())
	actorID := GetUserID(r.Context())
	s.writeAudit(audit.EventAuditEvidenceExported, audit.EntitySystem, "audit_anchoring", &actorID, map[string]string{"file": name})

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	if err := s.anchoring.WriteBundle(w); err != nil {
		// Headers are sent; the client sees a truncated zip.
		log.Printf("audit evidence bundle: %v", err)
	}
}
