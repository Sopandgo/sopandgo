package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_ArchiveSOP(t *testing.T) {
	env := testenv.New(t)

	editorID := "editor-archive-api"
	approverID := "approver-archive-api"
	env.SeedTestUser(t, editorID, "editor@archive-api.local", auth.RoleEditor)
	env.SeedTestUser(t, approverID, "approver@archive-api.local", auth.RoleApprover)
	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	approverToken, _ := auth.GenerateAccessToken(approverID, auth.RoleApprover)

	var ipCounter int
	doRequest := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var reqBody *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		} else {
			reqBody = bytes.NewReader([]byte{})
		}
		req, _ := http.NewRequest(method, path, reqBody)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}
	errorCode := func(rec *httptest.ResponseRecorder) string {
		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return body["error"]
	}
	listTotal := func(query string) int {
		rec := doRequest(http.MethodGet, "/api/sops"+query, editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list %s: %d %s", query, rec.Code, rec.Body.String())
		}
		var res struct {
			Total int `json:"total"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		return res.Total
	}

	sopID, err := env.SOPService.RegisterSOP("Archive API SOP", &editorID)
	if err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}
	if _, err := env.SOPService.RegisterSOP("Active API SOP", &editorID); err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}
	path := "/api/sops/" + sopID + "/archive"

	t.Run("editor cannot archive", func(t *testing.T) {
		rec := doRequest(http.MethodPost, path, editorToken, map[string]string{"reason": "Old"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("reason is required", func(t *testing.T) {
		rec := doRequest(http.MethodPost, path, approverToken, map[string]string{"reason": " "})
		if rec.Code != http.StatusBadRequest || errorCode(rec) != "invalid_archive_reason" {
			t.Fatalf("expected 400 invalid_archive_reason, got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unknown SOP", func(t *testing.T) {
		rec := doRequest(http.MethodPost, "/api/sops/nope/archive", approverToken, map[string]string{"reason": "Old"})
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("approver archives", func(t *testing.T) {
		rec := doRequest(http.MethodPost, path, approverToken, map[string]string{"reason": "Replaced"})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d %s", rec.Code, rec.Body.String())
		}
		rec = doRequest(http.MethodPost, path, approverToken, map[string]string{"reason": "Again"})
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 on second archive, got %d", rec.Code)
		}
	})

	t.Run("list filters and detail", func(t *testing.T) {
		if got := listTotal(""); got != 1 {
			t.Fatalf("default list total %d, want 1", got)
		}
		if got := listTotal("?archived=only"); got != 1 {
			t.Fatalf("archived=only total %d, want 1", got)
		}
		if got := listTotal("?archived=all"); got != 2 {
			t.Fatalf("archived=all total %d, want 2", got)
		}

		rec := doRequest(http.MethodGet, "/api/sops/"+sopID, editorToken, nil)
		var detail struct {
			Archive *struct {
				Reason string `json:"reason"`
			} `json:"archive"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &detail)
		if detail.Archive == nil || detail.Archive.Reason != "Replaced" {
			t.Fatalf("detail archive: %s", rec.Body.String())
		}
	})

	t.Run("archived SOP rejects new versions", func(t *testing.T) {
		rec := doRequest(http.MethodPost, "/api/sops/"+sopID, editorToken, map[string]string{
			"content":        "# Archived",
			"change_summary": "Try",
		})
		if rec.Code != http.StatusConflict || errorCode(rec) != "sop_archived" {
			t.Fatalf("expected 409 sop_archived, got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("editor cannot unarchive, approver can", func(t *testing.T) {
		rec := doRequest(http.MethodDelete, path, editorToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
		rec = doRequest(http.MethodDelete, path, approverToken, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d %s", rec.Code, rec.Body.String())
		}
		rec = doRequest(http.MethodDelete, path, approverToken, nil)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 when not archived, got %d", rec.Code)
		}
		if got := listTotal(""); got != 2 {
			t.Fatalf("default list total after unarchive %d, want 2", got)
		}
	})
}
