package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/branding"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func TestAPI_Branding(t *testing.T) {
	env := testenv.New(t)
	t.Cleanup(func() { i18n.SetAppName("") })

	adminID := "branding-admin"
	editorID := "branding-editor"
	env.SeedTestUser(t, adminID, "brand-admin@api.local", auth.RoleAdmin)
	env.SeedTestUser(t, editorID, "brand-editor@api.local", auth.RoleEditor)
	adminToken, err := auth.GenerateAccessToken(adminID, auth.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	editorToken, err := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	if err != nil {
		t.Fatal(err)
	}

	// Each request uses its own address so the suite stays under the per-IP rate limit.
	ipCounter := 0
	do := func(method, path, tok string, body []byte, contentType string, header map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		ipCounter++
		req.RemoteAddr = fmt.Sprintf("198.18.0.%d:1234", ipCounter%200+1)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range header {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}
	logoUpload := func(t *testing.T, tok string, content []byte) *httptest.ResponseRecorder {
		t.Helper()
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		part, _ := w.CreateFormFile("file", "logo.png")
		_, _ = part.Write(content)
		_ = w.Close()
		return do(http.MethodPost, "/api/admin/settings/branding/logo", tok, body.Bytes(), w.FormDataContentType(), nil)
	}
	decode := func(t *testing.T, rec *httptest.ResponseRecorder) branding.Branding {
		t.Helper()
		var b branding.Branding
		if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
			t.Fatalf("decode %s: %v", rec.Body.String(), err)
		}
		return b
	}

	t.Run("public GET works without a session", func(t *testing.T) {
		rec := do(http.MethodGet, "/api/branding", "", nil, "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
		}
		if b := decode(t, rec); b != (branding.Branding{}) {
			t.Fatalf("defaults: %+v", b)
		}
		if rec := do(http.MethodGet, "/api/branding/logo", "", nil, "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("logo before upload: %d", rec.Code)
		}
	})

	t.Run("editors cannot change branding", func(t *testing.T) {
		rec := do(http.MethodPut, "/api/admin/settings/branding", editorToken, []byte(`{"name":"x"}`), "application/json", nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("PUT as editor: %d", rec.Code)
		}
		if rec := logoUpload(t, editorToken, makeTestPNG(t, 10, 10)); rec.Code != http.StatusForbidden {
			t.Fatalf("logo as editor: %d", rec.Code)
		}
		if rec := do(http.MethodPut, "/api/admin/settings/branding", "", []byte(`{"name":"x"}`), "application/json", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("PUT without session: %d", rec.Code)
		}
	})

	t.Run("admin saves name and colours", func(t *testing.T) {
		rec := do(http.MethodPut, "/api/admin/settings/branding", adminToken,
			[]byte(`{"name":"Lab Handbook","primary_light":"#1d4ed8","primary_dark":""}`), "application/json", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d %s", rec.Code, rec.Body.String())
		}
		b := decode(t, rec)
		if b.Name != "Lab Handbook" || b.PrimaryLight != "#1d4ed8" || b.PrimaryDark != "" {
			t.Fatalf("saved: %+v", b)
		}
		if got := i18n.T("en", "mail.test.subject", nil); got != "Lab Handbook email test" {
			t.Fatalf("emails should use the name: %q", got)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		for body, code := range map[string]string{
			`{"primary_light":"#ffff00"}`:           "colour_contrast",
			`{"primary_dark":"blue"}`:               "colour_invalid",
			`{"name":"Line one\nBcc: evil@x.test"}`: "name_invalid",
		} {
			rec := do(http.MethodPut, "/api/admin/settings/branding", adminToken, []byte(body), "application/json", nil)
			if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte(code)) {
				t.Fatalf("%s: want 400 %s, got %d %s", body, code, rec.Code, rec.Body.String())
			}
		}
	})

	t.Run("logo upload, ETag, removal", func(t *testing.T) {
		if rec := logoUpload(t, adminToken, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)); rec.Code != http.StatusBadRequest {
			t.Fatalf("svg: %d %s", rec.Code, rec.Body.String())
		}

		rec := logoUpload(t, adminToken, makeTestPNG(t, 200, 80))
		if rec.Code != http.StatusOK {
			t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
		}
		b := decode(t, rec)
		if !b.HasLogo || b.LogoHash == "" {
			t.Fatalf("after upload: %+v", b)
		}

		for _, variant := range []string{"logo", "icon"} {
			rec := do(http.MethodGet, "/api/branding/logo?variant="+variant, "", nil, "", nil)
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" ||
				rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("%s: %d %v", variant, rec.Code, rec.Header())
			}
			etag := rec.Header().Get("ETag")
			if etag != `"`+b.LogoHash+`"` {
				t.Fatalf("%s etag %q", variant, etag)
			}
			again := do(http.MethodGet, "/api/branding/logo?variant="+variant, "", nil, "", map[string]string{"If-None-Match": etag})
			if again.Code != http.StatusNotModified {
				t.Fatalf("%s conditional: %d", variant, again.Code)
			}
		}
		if rec := do(http.MethodGet, "/api/branding/logo?variant=other", "", nil, "", nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("bad variant: %d", rec.Code)
		}

		if rec := do(http.MethodDelete, "/api/admin/settings/branding/logo", adminToken, nil, "", nil); rec.Code != http.StatusOK {
			t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
		}
		if rec := do(http.MethodDelete, "/api/admin/settings/branding/logo", adminToken, nil, "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("second delete: %d", rec.Code)
		}
		if rec := do(http.MethodGet, "/api/branding/logo", "", nil, "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("logo after delete: %d", rec.Code)
		}
	})

	t.Run("reset restores the defaults", func(t *testing.T) {
		if rec := do(http.MethodDelete, "/api/admin/settings/branding", editorToken, nil, "", nil); rec.Code != http.StatusForbidden {
			t.Fatalf("reset as editor: %d", rec.Code)
		}
		if rec := logoUpload(t, adminToken, makeTestPNG(t, 40, 40)); rec.Code != http.StatusOK {
			t.Fatalf("upload: %d", rec.Code)
		}
		rec := do(http.MethodDelete, "/api/admin/settings/branding", adminToken, nil, "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("reset: %d %s", rec.Code, rec.Body.String())
		}
		if b := decode(t, rec); b != (branding.Branding{}) {
			t.Fatalf("after reset: %+v", b)
		}
		if rec := do(http.MethodGet, "/api/branding/logo", "", nil, "", nil); rec.Code != http.StatusNotFound {
			t.Fatalf("logo after reset: %d", rec.Code)
		}
	})

	report, err := env.AuditLogger.VerifyEntireChain()
	if err != nil || report.Invalid != 0 {
		t.Fatalf("audit chain: %+v %v", report, err)
	}
}
