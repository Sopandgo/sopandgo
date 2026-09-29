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

func TestAPI_SOPs(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users
	editorID := "editor-sop-1"
	viewerID := "viewer-sop-1"

	env.SeedTestUser(t, editorID, "editor@sop.local", auth.RoleEditor)
	env.SeedTestUser(t, viewerID, "viewer@sop.local", auth.RoleViewer)

	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	var ipCounter int // <-- ADD THIS

	// Helper to fire HTTP requests
	doRequest := func(method, path, token string, body map[string]any) *httptest.ResponseRecorder {
		var reqBody *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		} else {
			reqBody = bytes.NewReader([]byte{})
		}

		req, _ := http.NewRequest(method, path, reqBody)

		// <-- ADD THESE TWO LINES -->
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	var createdSOPID string
	targetTitle := "Emergency Evacuation Protocol"

	// --- RUN TEST CASES ---

	t.Run("RBAC Rejection: Viewer cannot create SOP", func(t *testing.T) {
		payload := map[string]any{"title": "Hacker Protocol"}
		rec := doRequest(http.MethodPost, "/api/sops", viewerToken, payload)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for Viewer, got %d", rec.Code)
		}
	})

	t.Run("Create SOP (Happy Path)", func(t *testing.T) {
		payload := map[string]any{"title": targetTitle}
		rec := doRequest(http.MethodPost, "/api/sops", editorToken, payload)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res map[string]string
		json.NewDecoder(rec.Body).Decode(&res)

		createdSOPID = res["id"]
		if createdSOPID == "" {
			t.Fatal("Expected an ID to be returned, got empty string")
		}
	})

	t.Run("Create SOP (Bad Request)", func(t *testing.T) {
		// Send a deliberately malformed JSON string
		req, _ := http.NewRequest(http.MethodPost, "/api/sops", bytes.NewReader([]byte(`{ bad json }`)))
		req.Header.Set("Authorization", "Bearer "+editorToken)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for malformed JSON, got %d", rec.Code)
		}
	})

	t.Run("List SOPs", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/sops", viewerToken, nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		// Use an anonymous struct to catch the new wrapped payload
		var response struct {
			SOPs  []map[string]any `json:"sops"`
			Total int              `json:"total"`
		}

		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// We just created exactly 1 SOP
		if len(response.SOPs) != 1 {
			t.Fatalf("Expected 1 SOP in list, got %d. Total: %d", len(response.SOPs), response.Total)
		}

		if response.SOPs[0]["title"] != targetTitle {
			t.Errorf("Expected title %q, got %v", targetTitle, response.SOPs[0]["title"])
		}
	})

	t.Run("Get SOP By ID (Happy Path)", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s", createdSOPID)
		rec := doRequest(http.MethodGet, path, viewerToken, nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", rec.Code)
		}

		var sop map[string]any
		json.NewDecoder(rec.Body).Decode(&sop)

		if sop["id"] != createdSOPID {
			t.Errorf("Expected ID %q, got %v", createdSOPID, sop["id"])
		}
		if sop["title"] != targetTitle {
			t.Errorf("Expected title %q, got %v", targetTitle, sop["title"])
		}
	})

	t.Run("Get SOP By ID (Not Found)", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/sops/fake-uuid-1234", viewerToken, nil)

		// Your handler correctly maps sql.ErrNoRows to a 404 in the service layer
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("Favorite unfavorite and query params", func(t *testing.T) {
		favPath := fmt.Sprintf("/api/sops/%s/favorite", createdSOPID)
		rec := doRequest(http.MethodPost, favPath, viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST favorite: expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		rec = doRequest(http.MethodPost, favPath, viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("POST favorite idempotent: expected 200, got %d", rec.Code)
		}

		rec = doRequest(http.MethodGet, "/api/sops?favorites_only=true", viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list favorites_only: %d", rec.Code)
		}
		var favList struct {
			SOPs  []map[string]any `json:"sops"`
			Total int              `json:"total"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&favList); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if favList.Total != 1 || len(favList.SOPs) != 1 {
			t.Fatalf("favorites_only: want 1 sop, got total=%d len=%d", favList.Total, len(favList.SOPs))
		}
		if favList.SOPs[0]["is_favorite"] != true {
			t.Errorf("expected is_favorite true, got %v", favList.SOPs[0]["is_favorite"])
		}

		rec = doRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s", createdSOPID), viewerToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("get sop: %d", rec.Code)
		}
		var one map[string]any
		json.NewDecoder(rec.Body).Decode(&one)
		if one["is_favorite"] != true {
			t.Errorf("get sop expected is_favorite true for viewer, got %v", one["is_favorite"])
		}

		// Another user should not see this favorite on the same SOP
		rec = doRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s", createdSOPID), editorToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("get sop editor: %d", rec.Code)
		}
		json.NewDecoder(rec.Body).Decode(&one)
		if one["is_favorite"] == true {
			t.Error("editor should not see viewer favorite")
		}

		rec = doRequest(http.MethodDelete, favPath, viewerToken, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE favorite: expected 204, got %d", rec.Code)
		}
		rec = doRequest(http.MethodDelete, favPath, viewerToken, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE unfavorite idempotent: expected 204, got %d", rec.Code)
		}

		rec = doRequest(http.MethodGet, "/api/sops?favorites_only=true", viewerToken, nil)
		json.NewDecoder(rec.Body).Decode(&favList)
		if favList.Total != 0 {
			t.Errorf("after unfavorite favorites_only total want 0, got %d", favList.Total)
		}
	})
}
