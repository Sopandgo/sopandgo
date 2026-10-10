package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_SOPVersions(t *testing.T) {
	env := testenv.New(t)

	// --- SETUP USERS ---
	editorID := "editor-v-1"
	approverID := "approver-v-1"
	env.SeedTestUser(t, editorID, "editor@versions.local", auth.RoleEditor)
	env.SeedTestUser(t, approverID, "approver@versions.local", auth.RoleApprover)

	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	approverToken, _ := auth.GenerateAccessToken(approverID, auth.RoleApprover)

	var ipCounter int

	// Helper to send HTTP requests with JSON bodies
	doJSONRequest := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)

		// FIX: Spoof a unique IP address for every single request
		// to prevent the Rate Limiter from blocking our test suite!
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	// =========================================================================
	// LIFECYCLE 1: THE HAPPY PATH (Draft -> RC -> Published)
	// =========================================================================
	t.Run("Happy Path Lifecycle", func(t *testing.T) {
		// 1. Create isolated SOP container
		sopID, err := env.SOPService.RegisterSOP("Publishing Test SOP", &editorID)
		if err != nil {
			t.Fatalf("Failed to create base SOP: %v", err)
		}

		// 2. Create Draft
		testContent := "# Publishing Test\nThis will be published."
		path := fmt.Sprintf("/api/sops/%s", sopID)
		rec := doJSONRequest(http.MethodPost, path, editorToken, map[string]string{
			"content":        testContent,
			"change_summary": "Prepare the publishing test",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Draft creation failed: %d - %s", rec.Code, rec.Body.String())
		}

		var vResp struct {
			ID string `json:"id"`
		}
		json.NewDecoder(rec.Body).Decode(&vResp)
		vID := vResp.ID

		// 3. Verify Metadata is Draft
		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s", sopID, vID), editorToken, nil)
		var meta sop.SOPVersion
		json.NewDecoder(rec.Body).Decode(&meta)
		if meta.Status != sop.StateDraft {
			t.Errorf("Expected Draft, got %s", meta.Status)
		}

		// 4. Download content
		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/download", sopID, vID), editorToken, nil)
		bodyBytes, _ := io.ReadAll(rec.Body)
		if string(bodyBytes) != testContent {
			t.Errorf("Download mismatch. Expected %q, got %q", testContent, string(bodyBytes))
		}

		// 5. Promote to RC
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/promote", sopID, vID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Promotion failed: %d - %s", rec.Code, rec.Body.String())
		}

		// 6. Approve to Published
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/approve", sopID, vID), approverToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Approval failed: %d - %s", rec.Code, rec.Body.String())
		}

		// 7. Verify Latest Published Summary
		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/version-latest", sopID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Failed to fetch latest summary: %d", rec.Code)
		}
		var summary sop.SOPVersionSummary
		json.NewDecoder(rec.Body).Decode(&summary)
		if summary.ID != vID {
			t.Errorf("Latest summary ID mismatch: expected %s, got %s", vID, summary.ID)
		}
	})

	// =========================================================================
	// LIFECYCLE 2: THE REJECTION PATH (Draft -> RC -> Rejected)
	// =========================================================================
	t.Run("Rejection Path Lifecycle", func(t *testing.T) {
		// 1. Create a completely SEPARATE SOP container to avoid 500 state conflicts!
		rejectSopID, err := env.SOPService.RegisterSOP("Rejection Test SOP", &editorID)
		if err != nil {
			t.Fatalf("Failed to create rejection SOP: %v", err)
		}

		// 2. Create Draft
		path := fmt.Sprintf("/api/sops/%s", rejectSopID)
		rec := doJSONRequest(http.MethodPost, path, editorToken, map[string]string{
			"content":        "Bad Content",
			"change_summary": "This draft will be rejected",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Draft creation failed: %d - %s", rec.Code, rec.Body.String())
		}

		var vResp struct{ ID string }
		json.NewDecoder(rec.Body).Decode(&vResp)
		vID := vResp.ID

		// 3. Promote to RC
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/promote", rejectSopID, vID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Promotion failed: %d - %s", rec.Code, rec.Body.String())
		}

		// 4. Reject with Reason
		rejectPayload := map[string]string{"reason": "Does not meet quality standards"}
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/reject", rejectSopID, vID), approverToken, rejectPayload)
		if rec.Code != http.StatusOK {
			t.Fatalf("Rejection failed: %d - %s", rec.Code, rec.Body.String())
		}

		// 5. Verify State in Database
		v, err := env.SOPService.GetSOPVersionByID(vID)
		if err != nil {
			t.Fatalf("Failed to fetch rejected version: %v", err)
		}
		if v.Status != sop.StateRejected {
			t.Errorf("Expected state %s, got %s", sop.StateRejected, v.Status)
		}

		// 6. The reason is kept: on the version summary and in the audit chain
		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/summary", rejectSopID, vID), approverToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Summary failed: %d - %s", rec.Code, rec.Body.String())
		}
		var summary sop.SOPVersionSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
			t.Fatalf("Decode summary: %v", err)
		}
		if summary.Rejection == nil || summary.Rejection.Reason == nil || *summary.Rejection.Reason != "Does not meet quality standards" {
			t.Fatalf("Expected the rejection reason on the summary, got %+v", summary.Rejection)
		}
		if summary.Rejection.ActorUserID != approverID || summary.Rejection.ActorName == "" {
			t.Errorf("Expected the approver as rejecting actor, got %+v", summary.Rejection)
		}
		var auditReasons int
		if err := env.Store.DB.QueryRow(
			`SELECT COUNT(*) FROM audit_events WHERE entity_id = ? AND payload LIKE '%Does not meet quality standards%'`, vID,
		).Scan(&auditReasons); err != nil || auditReasons != 1 {
			t.Errorf("Expected the reason in one audit event, got %d (%v)", auditReasons, err)
		}

		// 7. A rejected version cannot be rejected again
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/%s/reject", rejectSopID, vID), approverToken, rejectPayload)
		if rec.Code != http.StatusConflict {
			t.Errorf("Expected 409 rejecting a rejected version, got %d", rec.Code)
		}
	})

	// =========================================================================
	// EDGE CASES & ERROR HANDLING
	// =========================================================================
	t.Run("Edge Cases", func(t *testing.T) {
		sopID, _ := env.SOPService.RegisterSOP("Edge Case SOP", &editorID)

		// 1. Register with empty content
		rec := doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s", sopID), editorToken, map[string]string{"content": ""})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty content, got %d", rec.Code)
		}

		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s", sopID), editorToken, map[string]string{
			"content":        "# Missing summary",
			"change_summary": "   ",
		})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty change summary, got %d", rec.Code)
		}

		// 2. Promote an invalid version ID
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/fake-id/promote", sopID), editorToken, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for fake version promotion, got %d", rec.Code)
		}

		// 3. Reject without a reason
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s/versions/fake-id/reject", sopID), approverToken, map[string]string{"reason": ""})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing rejection reason, got %d", rec.Code)
		}

		// 4. Markdown policy (e.g. raw HTML) returns 400 with structured JSON
		rec = doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s", sopID), editorToken, map[string]string{
			"content":        "# x\n\n<div>nope</div>\n",
			"change_summary": "Should be rejected by markdown policy",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 for disallowed markdown, got %d: %s", rec.Code, rec.Body.String())
		}
		var policyResp struct {
			Error  string `json:"error"`
			Code   string `json:"code"`
			Detail string `json:"detail"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&policyResp); err != nil {
			t.Fatalf("decode policy error: %v", err)
		}
		if policyResp.Code != "raw_html" {
			t.Errorf("Expected raw_html code, got %+v", policyResp)
		}
	})

	t.Run("Version Integrity", func(t *testing.T) {
		sopID, err := env.SOPService.RegisterSOP("Integrity Test SOP", &editorID)
		if err != nil {
			t.Fatalf("Failed to create base SOP: %v", err)
		}
		otherSOPID, err := env.SOPService.RegisterSOP("Other SOP", &editorID)
		if err != nil {
			t.Fatalf("Failed to create other SOP: %v", err)
		}

		rec := doJSONRequest(http.MethodPost, fmt.Sprintf("/api/sops/%s", sopID), editorToken, map[string]string{
			"content":        "# Integrity\nBody.",
			"change_summary": "Integrity test",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("Draft creation failed: %d - %s", rec.Code, rec.Body.String())
		}
		var vResp struct {
			ID string `json:"id"`
		}
		json.NewDecoder(rec.Body).Decode(&vResp)

		decodeErr := func(rec *httptest.ResponseRecorder) string {
			var body struct {
				Error string `json:"error"`
			}
			json.NewDecoder(rec.Body).Decode(&body)
			return body.Error
		}

		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/integrity", sopID, vResp.ID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d - %s", rec.Code, rec.Body.String())
		}
		var ok map[string]bool
		json.NewDecoder(rec.Body).Decode(&ok)
		if !ok["hash_valid"] {
			t.Error("Expected hash_valid=true")
		}

		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/integrity", otherSOPID, vResp.ID), editorToken, nil)
		if rec.Code != http.StatusNotFound || decodeErr(rec) != "version_not_found" {
			t.Errorf("Expected 404 version_not_found for SOP mismatch, got %d", rec.Code)
		}

		path, err := env.SOPService.GetVersionPath(vResp.ID)
		if err != nil {
			t.Fatalf("GetVersionPath: %v", err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove version file: %v", err)
		}
		rec = doJSONRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/versions/%s/integrity", sopID, vResp.ID), editorToken, nil)
		if rec.Code != http.StatusNotFound || decodeErr(rec) != "file_missing" {
			t.Errorf("Expected 404 file_missing, got %d", rec.Code)
		}
	})
}
