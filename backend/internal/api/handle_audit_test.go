package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_AdminAuditLogs(t *testing.T) {
	env := testenv.New(t)

	// 1. Setup Users & Tokens
	adminID := "admin-audit-1"
	viewerID := "viewer-audit-1"
	targetID := "target-audit-1"

	env.SeedTestUser(t, adminID, "admin@audit.local", auth.RoleAdmin)
	env.SeedTestUser(t, viewerID, "viewer@audit.local", auth.RoleViewer)
	env.SeedTestUser(t, targetID, "target@audit.local", auth.RoleViewer)

	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	// 2. Generate some guaranteed Audit Logs!
	// We will change the target user's role a few times to generate "user_role_updated" events.
	env.AuthService.UpdateUserRole(targetID, auth.RoleEditor, &adminID)
	env.AuthService.UpdateUserRole(targetID, auth.RoleViewer, &adminID)
	env.AuthService.UpdateUserRole(targetID, auth.RoleEditor, &adminID)
	sopID, _ := env.SOPService.RegisterSOP("Audit Filter SOP", &adminID)

	var ipCounter int

	// Helper to fire standard HTTP requests
	doRequest := func(method, path, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, nil)
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ipCounter)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	// Define the expected JSON response shape
	type AuditResponse struct {
		Events []map[string]any `json:"events"`
		Total  int              `json:"total"`
	}

	// --- RUN TEST CASES ---

	t.Run("RBAC Rejection: Viewer accessing audit logs", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/audit-logs", viewerToken)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for Viewer, got %d", rec.Code)
		}
	})

	t.Run("Fetch All Logs (No params)", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/audit-logs", adminToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		// We generated 3 role updates, plus the system might have generated others
		// depending on your testenv setup, so total should be >= 3
		if res.Total < 3 {
			t.Errorf("Expected at least 3 total logs, got %d", res.Total)
		}
		if len(res.Events) < 3 {
			t.Errorf("Expected at least 3 events in array, got %d", len(res.Events))
		}
	})

	t.Run("Pagination: Limit and Offset", func(t *testing.T) {
		// We ask for exactly 2 items
		path := "/api/admin/audit-logs?limit=2&offset=0"
		rec := doRequest(http.MethodGet, path, adminToken)

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		// The array length should be strictly bound by our limit
		if len(res.Events) != 2 {
			t.Errorf("Expected exactly 2 events returned due to limit, got %d", len(res.Events))
		}
		// But the Total count should still represent the full database count!
		if res.Total < 3 {
			t.Errorf("Expected total count to remain accurate (>= 3), got %d", res.Total)
		}
	})

	t.Run("Filtering: Specific Event Type", func(t *testing.T) {
		// We specifically ask for only the role updates we triggered
		path := "/api/admin/audit-logs?type=user_role_updated"
		rec := doRequest(http.MethodGet, path, adminToken)

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		// We triggered exactly 3 of these
		if len(res.Events) != 3 {
			t.Errorf("Expected exactly 3 'user_role_updated' events, got %d", len(res.Events))
		}

		// Verify that the filtering actually worked on the data returned
		for _, event := range res.Events {
			// FIX: Check "event_type" instead of "action".
			// If your struct uses a different JSON tag like "type" or "action_name",
			// just swap it here!
			if event["event_type"] != "user_role_updated" {
				t.Errorf("Filter failed: returned an event of type %v", event["event_type"])
			}
		}
	})

	t.Run("Filtering: Entity Type", func(t *testing.T) {
		path := "/api/admin/audit-logs?entity_type=user"
		rec := doRequest(http.MethodGet, path, adminToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		if len(res.Events) == 0 {
			t.Fatalf("Expected at least one event for entity_type=user")
		}
		for _, event := range res.Events {
			if event["entity_type"] != "user" {
				t.Errorf("Filter failed: returned entity_type %v", event["entity_type"])
			}
		}
	})

	t.Run("Filtering: Actor User ID", func(t *testing.T) {
		path := "/api/admin/audit-logs?actor_user_id=" + adminID + "&type=user_role_updated"
		rec := doRequest(http.MethodGet, path, adminToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		if len(res.Events) != 3 {
			t.Errorf("Expected exactly 3 role updates by admin, got %d", len(res.Events))
		}
		for _, event := range res.Events {
			if event["actor_user_id"] != adminID {
				t.Errorf("Filter failed: returned actor_user_id %v", event["actor_user_id"])
			}
		}
	})

	t.Run("Filtering: SOP ID", func(t *testing.T) {
		path := "/api/admin/audit-logs?sop_id=" + sopID
		rec := doRequest(http.MethodGet, path, adminToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var res AuditResponse
		json.NewDecoder(rec.Body).Decode(&res)

		if len(res.Events) == 0 {
			t.Fatalf("Expected at least one event for sop_id filter")
		}
		for _, event := range res.Events {
			entityID, _ := event["entity_id"].(string)
			payload, _ := event["payload"].(string)
			if entityID != sopID && payload != "" && !strings.Contains(payload, sopID) {
				t.Errorf("SOP filter returned unrelated event id=%v payload=%v", entityID, payload)
			}
		}
	})

	t.Run("Pagination: Invalid limit returns 400", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/audit-logs?limit=abc", adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Pagination: Invalid offset returns 400", func(t *testing.T) {
		rec := doRequest(http.MethodGet, "/api/admin/audit-logs?offset=-1", adminToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}
