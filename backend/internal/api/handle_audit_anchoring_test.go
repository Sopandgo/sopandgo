package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/anchor"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/objectstore"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// The service's flows are tested in internal/anchor; this covers routing,
// access, and how service errors become responses.
func TestAPI_AuditAnchoring(t *testing.T) {
	env := testenv.New(t)
	adminID := "anchor-admin"
	env.SeedTestUser(t, adminID, "anchor-admin@api.local", auth.RoleAdmin)
	adminToken, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	viewerID := "anchor-viewer"
	env.SeedTestUser(t, viewerID, "anchor-viewer@api.local", auth.RoleViewer)
	viewerToken, _ := auth.GenerateAccessToken(viewerID, auth.RoleViewer)

	var ip int
	do := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, path, r)
		ip++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ip)
		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	// Not wired: endpoints say so, and the integrity scan has no anchoring section.
	if rec := do("GET", "/api/admin/audit-anchoring", adminToken, nil); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unwired GET = %d", rec.Code)
	}
	rec := do("GET", "/api/admin/integrity", adminToken, nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `"anchoring"`) {
		t.Fatalf("integrity = %d %s", rec.Code, rec.Body.String())
	}

	noS3 := func(context.Context) (objectstore.Store, error) { return nil, backup.ErrS3NotConfigured }
	svc := anchor.NewService(anchor.NewStore(env.Store.DB, []byte("01234567890123456789012345678901")), env.AuditLogger, noS3, nil)
	if err := svc.Load(); err != nil {
		t.Fatal(err)
	}
	env.API.SetAuditAnchoring(svc)

	if rec := do("GET", "/api/admin/audit-anchoring", viewerToken, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("viewer GET = %d", rec.Code)
	}
	rec = do("GET", "/api/admin/audit-anchoring", adminToken, nil)
	var st anchor.Status
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &st) != nil || st.Enabled {
		t.Fatalf("GET = %d %s", rec.Code, rec.Body.String())
	}

	// Invalid input is a 400; a missing S3 bucket is a precondition.
	if rec := do("PUT", "/api/admin/audit-anchoring", adminToken, map[string]any{"enabled": true, "tsa_url": "ftp://x"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad URL = %d %s", rec.Code, rec.Body.String())
	}
	rec = do("PUT", "/api/admin/audit-anchoring", adminToken, map[string]any{"enabled": true, "tsa_url": "http://tsa.example/tsr"})
	if rec.Code != http.StatusPreconditionFailed || !strings.Contains(rec.Body.String(), "S3") {
		t.Fatalf("no S3 = %d %s", rec.Code, rec.Body.String())
	}

	if rec := do("POST", "/api/admin/audit-anchoring/run", adminToken, nil); rec.Code != http.StatusConflict {
		t.Fatalf("run while off = %d", rec.Code)
	}

	rec = do("GET", "/api/admin/audit-anchoring/bundle", adminToken, nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" || !bytes.HasPrefix(rec.Body.Bytes(), []byte("PK")) {
		t.Fatalf("bundle = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var exported int
	env.Store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'audit_evidence_exported'`).Scan(&exported)
	if exported != 1 {
		t.Fatalf("evidence export audited %d times", exported)
	}
}
