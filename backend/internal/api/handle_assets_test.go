package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_Assets(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users & Tokens
	authorID := "author-1"
	viewerID := "viewer-1"

	env.SeedTestUser(t, authorID, "author@api.local", auth.RoleEditor)
	env.SeedTestUser(t, viewerID, "viewer@api.local", auth.RoleViewer)

	authorToken, _ := auth.GenerateAccessToken(authorID, auth.RoleEditor)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	// 2. Setup Base SOP
	sopID, err := env.SOPService.RegisterSOP("Asset Testing SOP", &authorID)
	if err != nil {
		t.Fatalf("Failed to setup SOP: %v", err)
	}

	// Helper to fire standard HTTP requests
	doRequest := func(method, path, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	var uploadedAssetID string
	filePayload := []byte("fake image data payload")
	fileName := "diagram.png"

	// --- RUN TEST CASES ---

	t.Run("RBAC Rejection: Viewer Uploading Asset", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets", sopID)

		// Viewers don't have ScopeSOPWrite, so this MUST fail before the handler even parses the body.
		rec := doRequest(http.MethodPost, path, viewerToken)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for viewer upload, got %d", rec.Code)
		}
	})

	t.Run("Upload Asset (Multipart Form)", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets", sopID)

		// 1. Construct the Multipart Form Data
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create the file part matching `r.FormFile("file")`
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write(filePayload)
		writer.Close() // Must close to finalize the boundary!

		// 2. Create and execute the request
		req, _ := http.NewRequest(http.MethodPost, path, body)
		req.Header.Set("Authorization", "Bearer "+authorToken)
		req.Header.Set("Content-Type", writer.FormDataContentType()) // CRITICAL!

		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		// 3. Extract the new Asset ID for the remaining tests
		var response map[string]string
		json.NewDecoder(rec.Body).Decode(&response)
		uploadedAssetID = response["id"]
		if uploadedAssetID == "" {
			t.Fatal("Expected an asset ID in the response")
		}
	})

	t.Run("List Assets", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets", sopID)
		rec := doRequest(http.MethodGet, path, viewerToken) // Viewers CAN list

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var assets []map[string]any
		json.NewDecoder(rec.Body).Decode(&assets)
		if len(assets) != 1 {
			t.Errorf("Expected 1 asset, got %d", len(assets))
		}
	})

	t.Run("Get Asset Metadata", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets/%s", sopID, uploadedAssetID)
		rec := doRequest(http.MethodGet, path, viewerToken)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var asset map[string]any
		json.NewDecoder(rec.Body).Decode(&asset)

		// FIX: Use "content_path" (or whatever JSON tag your struct uses)
		// and check if it ends with our file name.
		contentPath, ok := asset["content_path"].(string)

		// If your struct uses a different tag (like "file_name" or "path"),
		// change "content_path" above to match it!
		if !ok || !strings.HasSuffix(contentPath, fileName) {
			t.Errorf("Expected content_path to end with %q, got %v", fileName, asset["content_path"])
		}
	})

	t.Run("Download Asset (Stream)", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets/%s/download", sopID, uploadedAssetID)
		rec := doRequest(http.MethodGet, path, viewerToken)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		// Verify the binary payload streamed back matches what we uploaded perfectly
		downloadedBytes, _ := io.ReadAll(rec.Body)
		if !bytes.Equal(filePayload, downloadedBytes) {
			t.Errorf("Downloaded bytes do not match uploaded bytes!")
		}
	})

	t.Run("Check Asset Integrity", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets/%s/integrity", sopID, uploadedAssetID)
		rec := doRequest(http.MethodGet, path, authorToken)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", rec.Code)
		}

		var response map[string]bool
		json.NewDecoder(rec.Body).Decode(&response)
		if response["hash_valid"] != true {
			t.Error("Expected hash_valid to be true")
		}
	})

	t.Run("Unhappy Path: Download non-existent asset", func(t *testing.T) {
		path := fmt.Sprintf("/api/sops/%s/assets/fake-id-123/download", sopID)
		rec := doRequest(http.MethodGet, path, viewerToken)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", rec.Code)
		}
	})

	t.Run("Asset Integrity: wrong SOP and missing file", func(t *testing.T) {
		otherSOPID, err := env.SOPService.RegisterSOP("Other Asset SOP", &authorID)
		if err != nil {
			t.Fatalf("Failed to setup other SOP: %v", err)
		}
		decodeErr := func(rec *httptest.ResponseRecorder) string {
			var body struct {
				Error string `json:"error"`
			}
			json.NewDecoder(rec.Body).Decode(&body)
			return body.Error
		}

		rec := doRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/assets/%s/integrity", otherSOPID, uploadedAssetID), authorToken)
		if rec.Code != http.StatusNotFound || decodeErr(rec) != "asset_not_found" {
			t.Errorf("Expected 404 asset_not_found for SOP mismatch, got %d", rec.Code)
		}

		path, err := env.SOPService.GetAssetPath(uploadedAssetID)
		if err != nil {
			t.Fatalf("GetAssetPath: %v", err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove asset file: %v", err)
		}
		rec = doRequest(http.MethodGet, fmt.Sprintf("/api/sops/%s/assets/%s/integrity", sopID, uploadedAssetID), authorToken)
		if rec.Code != http.StatusNotFound || decodeErr(rec) != "file_missing" {
			t.Errorf("Expected 404 file_missing, got %d", rec.Code)
		}
	})
}
