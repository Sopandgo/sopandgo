package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_Health(t *testing.T) {
	env := testenv.New(t)

	// 1. Construct the Request
	req, _ := http.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	// 2. Fire it through the router
	env.API.ServeHTTP(rec, req)

	// 3. Assert HTTP Status
	if rec.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rec.Code)
	}

	// 4. Assert Headers
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	// 5. Assert JSON Body
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response["status"] != "available" {
		t.Errorf("Expected status 'available', got %q", response["status"])
	}
}
