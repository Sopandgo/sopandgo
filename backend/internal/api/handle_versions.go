package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sopandgo/sopandgo/backend/internal/markdown"
	"github.com/sopandgo/sopandgo/backend/internal/notify"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
)

type versionIDResponse struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

func (s *Server) handleRegisterSOPVersion(w http.ResponseWriter, r *http.Request) {
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	var req struct {
		Content       string `json:"content"`
		ChangeSummary string `json:"change_summary"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	id, versionNum, err := s.sopService.RegisterSOPVersion(
		sopId,
		req.Content,
		req.ChangeSummary,
		actorID,
	)
	if err != nil {
		var pe *markdown.PolicyError
		if errors.As(err, &pe) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":  "markdown policy violation",
				"code":   pe.Code,
				"detail": pe.Detail,
			})
			return
		}

		msg := err.Error()
		switch {
		case errors.Is(err, sop.ErrSOPArchived):
			writeSOPArchived(w)
			return
		case strings.Contains(msg, "referenced asset not found:"),
			strings.Contains(msg, "invalid asset URL encoding:"):
			writeJSONError(w, http.StatusBadRequest, "invalid_asset_reference", msg)
			return
		case errors.Is(err, sop.ErrChangeSummaryRequired), errors.Is(err, sop.ErrChangeSummaryTooLong):
			writeJSONError(w, http.StatusBadRequest, "invalid_change_summary", err.Error())
			return
		case strings.HasPrefix(msg, "conflict:"):
			d := strings.TrimSpace(strings.TrimPrefix(msg, "conflict:"))
			writeJSONError(w, http.StatusConflict, "conflict", d)
			return
		case strings.Contains(msg, "not found"):
			http.Error(w, "SOP not found", http.StatusNotFound)
			return
		}

		log.Printf("ERROR: create version for SOP %s failed: %v", sopId, err)
		http.Error(w, "Failed to create version", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(versionIDResponse{
		ID:      id,
		Version: versionNum,
	})
}

func (s *Server) handleGetSOPVersionSummaryByID(w http.ResponseWriter, r *http.Request) {
	// Currently ignored, could be reused to add more granular access
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionId := r.PathValue("sopVersionID")
	if sopVersionId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	summary, err := s.sopService.GetSOPVersionSummaryByID(sopVersionId)
	if err != nil {
		http.Error(w, "Failed to retrieve SOP summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleListSOPVersions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("sopID")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	versions, err := s.sopService.GetSOPVersionsBySOPID(id)
	if err != nil {
		http.Error(w, "Failed to list versions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

func (s *Server) handleGetSOPVersionByID(w http.ResponseWriter, r *http.Request) {
	// Currently ignored, could be reused to add more granular access
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionId := r.PathValue("sopVersionID")
	if sopVersionId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	// Returns just the DB metadata, no file content
	v, err := s.sopService.GetSOPVersionByID(sopVersionId)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleGetSOPVersionSummaryLatest(w http.ResponseWriter, r *http.Request) {
	// Currently ignored, could be reused to add more granular access
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionId, err := s.sopService.GetSOPVersionIDLatestPublished(sopId)
	if err != nil {
		http.Error(w, "Failed to find latest SOP version ID", http.StatusNoContent)
		return
	}

	summary, err := s.sopService.GetSOPVersionSummaryByID(sopVersionId)
	if err != nil {
		http.Error(w, "Failed to retrieve SOP summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleCheckVersionIntegrity(w http.ResponseWriter, r *http.Request) {
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionId := r.PathValue("sopVersionID")
	if sopVersionId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	version, err := s.sopService.GetSOPVersionByID(sopVersionId)
	if err != nil || version.SOPID != sopId {
		writeJSONError(w, http.StatusNotFound, "version_not_found", "Version not found for this SOP.")
		return
	}

	valid, err := s.sopService.VerifyVersionIntegrity(sopVersionId)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusNotFound, "file_missing", "File missing on disk.")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "integrity_check_failed", "Integrity check failed.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"hash_valid": valid})
}

// handleDownloadSOPVersion streams the raw markdown file.
// Supports GET (browser download) and POST (API fetch).
func (s *Server) handleDownloadSOPVersion(w http.ResponseWriter, r *http.Request) {
	// Currently ignored, could be reused to add more granular access
	sopId := r.PathValue("sopID")
	if sopId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionId := r.PathValue("sopVersionID")
	if sopVersionId == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	absPath, err := s.sopService.GetVersionPath(sopVersionId)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/markdown")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=sop_version_%s.md", sopVersionId))

	http.ServeFile(w, r, absPath)
}

func (s *Server) handleDownloadSOPVersionPDF(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	sopVersionID := r.PathValue("sopVersionID")
	if sopVersionID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	stage := r.URL.Query().Get("stage")
	absPath, generatorVersion, err := s.sopService.GetVersionPDFArtifactPath(sopVersionID, stage)
	if err != nil {
		if errors.Is(err, sop.ErrPDFExportDisabled) {
			http.Error(w, "PDF export is disabled by operator", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "PDF artifact not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=sop_version_%s_g%s.pdf", sopVersionID, generatorVersion))
	http.ServeFile(w, r, absPath)
}

// writeLifecycleError answers a failed promote, approve or reject. The SOP service
// owns the lifecycle rules; the handler only maps its errors to status codes.
func writeLifecycleError(w http.ResponseWriter, err error, action, versionID, failMsg string) {
	msg := err.Error()
	switch {
	case errors.Is(err, sop.ErrSOPArchived):
		writeSOPArchived(w)
	case errors.Is(err, sop.ErrRejectReasonRequired), errors.Is(err, sop.ErrRejectReasonTooLong):
		http.Error(w, msg, http.StatusBadRequest)
	case strings.HasPrefix(msg, "conflict:"):
		http.Error(w, sentence(strings.TrimSpace(strings.TrimPrefix(msg, "conflict:"))), http.StatusConflict)
	case strings.Contains(msg, "not found"):
		http.Error(w, "Version not found", http.StatusNotFound)
	default:
		log.Printf("ERROR: %s version %s failed: %v", action, versionID, err)
		http.Error(w, failMsg, http.StatusInternalServerError)
	}
}

// sentence capitalizes the first letter of a service error detail.
func sentence(detail string) string {
	r, size := utf8.DecodeRuneInString(detail)
	if r == utf8.RuneError {
		return detail
	}
	return string(unicode.ToUpper(r)) + detail[size:]
}

func (s *Server) handlePromoteSOPVersion(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("sopVersionID")
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())
	if err := s.sopService.PromoteSOPVersion(versionID, actorID); err != nil {
		writeLifecycleError(w, err, "promote", versionID, "Failed to promote version")
		return
	}

	s.notifyLifecycle(notify.EventSOPRC, versionID, actorID, "")

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleApproveSOPVersion(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("sopVersionID")
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}

	actorID := GetUserID(r.Context())

	// The service publishes, records the approver signature and supersedes the old version.
	ackID, err := s.sopService.ApproveSOPVersion(versionID, actorID)
	if err != nil {
		writeLifecycleError(w, err, "approve", versionID, "Failed to approve and publish SOP")
		return
	}

	s.notifyVersionPublished(versionID, actorID)
	s.notifyLifecycle(notify.EventSOPPublished, versionID, actorID, "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"id": ackID,
	})
}

func (s *Server) handleRejectSOPVersion(w http.ResponseWriter, r *http.Request) {
	versionID := r.PathValue("sopVersionID")
	if versionID == "" {
		http.Error(w, "missing version id", http.StatusBadRequest)
		return
	}

	// Parse the reason from the JSON body
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		http.Error(w, "invalid request or missing reason", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(req.Reason)

	actorID := GetUserID(r.Context())
	if err := s.sopService.RejectSOPVersion(versionID, actorID, reason); err != nil {
		writeLifecycleError(w, err, "reject", versionID, "Failed to reject version")
		return
	}

	s.notifyLifecycle(notify.EventSOPRejected, versionID, actorID, reason)

	w.WriteHeader(http.StatusOK)
}
