package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
}
