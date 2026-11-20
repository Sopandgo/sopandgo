package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/errreport"
)

// errorReportingEntityID names the error reporting settings in the audit log.
const errorReportingEntityID = "error_reporting"

// SetErrorReporting connects error reporting: the settings routes and the
// middleware that recovers panics and reports server errors. The middleware
// wraps everything, the demo guard included, so a panic anywhere is caught.
// Call it once, before serving.
func (s *Server) SetErrorReporting(svc *errreport.Service) {
	s.errReport = svc
	s.handler = svc.Middleware(s.handler)
}

func (s *Server) errorReportingOr503(w http.ResponseWriter) bool {
	if s.errReport == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "error_reporting_unavailable", "error reporting is not available on this server")
		return false
	}
	return true
}

func (s *Server) handleGetErrorReportingConfig(w http.ResponseWriter, r *http.Request) {
	cfg := errreport.PublicConfig{}
	if s.errReport != nil {
		cfg = s.errReport.PublicConfig()
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleAdminGetErrorReporting(w http.ResponseWriter, r *http.Request) {
	if !s.errorReportingOr503(w) {
		return
	}
	st, err := s.errReport.Get()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "load_failed", "failed to load error reporting settings")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleAdminPutErrorReporting(w http.ResponseWriter, r *http.Request) {
	if !s.errorReportingOr503(w) {
		return
	}
	var req struct {
		Enabled        bool   `json:"enabled"`
		DSN            string `json:"dsn"`
		Environment    string `json:"environment"`
		BrowserEnabled bool   `json:"browser_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	st, err := s.errReport.Save(errreport.SaveInput{
		Enabled:        req.Enabled,
		DSN:            req.DSN,
		Environment:    req.Environment,
		BrowserEnabled: req.BrowserEnabled,
	})
	if err != nil {
		writeErrorReportingSaveError(w, err)
		return
	}
	actorID := GetUserID(r.Context())
	host, project := errreport.DSNTarget(st.DSN)
	// The audit log names the endpoint, never the full DSN.
	s.writeAudit(audit.EventErrorReportingUpdated, audit.EntityIntegration, errorReportingEntityID, &actorID, map[string]any{
		"enabled":         st.Enabled,
		"environment":     st.Environment,
		"browser_enabled": st.BrowserEnabled,
		"dsn_host":        host,
		"dsn_project":     project,
	})
	w.WriteHeader(http.StatusNoContent)
}

func writeErrorReportingSaveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errreport.ErrDSNRequired):
		writeKindJSON(w, err, "dsn_required", err.Error())
	case errors.Is(err, errreport.ErrDSNInvalid):
		writeKindJSON(w, err, "dsn_invalid", err.Error())
	case errors.Is(err, errreport.ErrEnvironmentInvalid):
		writeKindJSON(w, err, "environment_invalid", err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "save_failed", "failed to save error reporting settings")
	}
}

func (s *Server) handleAdminPostErrorReportingTest(w http.ResponseWriter, r *http.Request) {
	if !s.errorReportingOr503(w) {
		return
	}
	id, err := s.errReport.SendTest()
	if errors.Is(err, errreport.ErrNotEnabled) {
		writeKindJSON(w, err, "not_enabled", "Turn error reporting on and save before sending a test event.")
		return
	}
	result := "delivered"
	if err != nil {
		result = "failed"
	}
	actorID := GetUserID(r.Context())
	host, _ := errreport.DSNTarget(s.errReport.PublicConfig().DSN)
	s.writeAudit(audit.EventErrorReportingTestSent, audit.EntityIntegration, errorReportingEntityID, &actorID, map[string]any{
		"result":   result,
		"dsn_host": host,
	})
	if err != nil {
		// 502, like the other test buttons: the admin's endpoint failed. The
		// error reporting middleware does not report 502s.
		writeJSONError(w, http.StatusBadGateway, "delivery_failed", "The test event was not delivered. Check the DSN and that SOPandGO can reach the server.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"event_id": id})
}
