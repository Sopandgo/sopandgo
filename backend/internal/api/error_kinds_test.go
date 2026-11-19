package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// TestAPI_SOPErrorStatuses pins the status codes, JSON error codes and the
// detail text the frontend reads for SOP routes whose failures come from the
// SOP module's error kinds.
func TestAPI_SOPErrorStatuses(t *testing.T) {
	env := testenv.New(t)

	editorID, approverID, readerID := "kinds-editor", "kinds-approver", "kinds-reader"
	env.SeedTestUser(t, editorID, "editor@kinds.local", auth.RoleEditor)
	env.SeedTestUser(t, approverID, "approver@kinds.local", auth.RoleApprover)
	env.SeedTestUser(t, readerID, "reader@kinds.local", auth.RoleViewer)
	editorToken, _ := auth.GenerateAccessToken(editorID, auth.RoleEditor)
	approverToken, _ := auth.GenerateAccessToken(approverID, auth.RoleApprover)
	readerToken, _ := auth.GenerateAccessToken(readerID, auth.RoleViewer)

	var ip int
	serve := func(req *http.Request, token string) *httptest.ResponseRecorder {
		ip++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ip%250+1)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}
	doJSON := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		return serve(req, token)
	}
	upload := func(path, token, filename string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("file", filename)
		_, _ = part.Write([]byte("asset bytes"))
		_ = w.Close()
		req := httptest.NewRequest(http.MethodPost, path, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		return serve(req, token)
	}
	jsonError := func(t *testing.T, rec *httptest.ResponseRecorder) (code, detail string) {
		t.Helper()
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("want a JSON error body, got %q", rec.Body.String())
		}
		return body["error"], body["detail"]
	}
	want := func(t *testing.T, rec *httptest.ResponseRecorder, status int) {
		t.Helper()
		if rec.Code != status {
			t.Fatalf("status = %d, want %d; body %q", rec.Code, status, rec.Body.String())
		}
	}

	mustNoErr := func(what string, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}
	published := func(title string) (sopID, versionID string) {
		t.Helper()
		sopID, err := env.SOPService.RegisterSOP(title, &editorID)
		mustNoErr("register SOP", err)
		versionID, _, err = env.SOPService.RegisterSOPVersion(sopID, "# "+title, "First version", editorID)
		mustNoErr("register version", err)
		mustNoErr("promote", env.SOPService.PromoteSOPVersion(versionID, approverID))
		_, err = env.SOPService.ApproveSOPVersion(versionID, approverID)
		mustNoErr("approve", err)
		return sopID, versionID
	}

	sopID, publishedID := published("Kinds Centrifuge")
	otherSOPID, otherVersionID := published("Kinds Autoclave")
	_ = otherSOPID

	t.Run("version create with a missing asset is 400 invalid_asset_reference", func(t *testing.T) {
		rec := doJSON(http.MethodPost, "/api/sops/"+sopID, editorToken, map[string]string{
			"content":        "See ![diagram](assets/missing-diagram.png)",
			"change_summary": "Add diagram",
		})
		want(t, rec, http.StatusBadRequest)
		code, detail := jsonError(t, rec)
		if code != "invalid_asset_reference" || !strings.HasPrefix(detail, "referenced asset not found: assets/missing-diagram.png") {
			t.Fatalf("got %q %q", code, detail)
		}
	})

	t.Run("version create for an unknown SOP is 404", func(t *testing.T) {
		rec := doJSON(http.MethodPost, "/api/sops/no-such-sop", editorToken, map[string]string{
			"content": "# Missing", "change_summary": "First",
		})
		want(t, rec, http.StatusNotFound)
	})

	t.Run("version create while a release candidate is pending is 409 conflict", func(t *testing.T) {
		rcSOP, err := env.SOPService.RegisterSOP("Kinds Pipette", &editorID)
		mustNoErr("register SOP", err)
		v, _, err := env.SOPService.RegisterSOPVersion(rcSOP, "# Pipette", "First", editorID)
		mustNoErr("register version", err)
		mustNoErr("promote", env.SOPService.PromoteSOPVersion(v, approverID))

		rec := doJSON(http.MethodPost, "/api/sops/"+rcSOP, editorToken, map[string]string{
			"content": "# Pipette v2", "change_summary": "Second",
		})
		want(t, rec, http.StatusConflict)
		code, detail := jsonError(t, rec)
		if code != "conflict" || !strings.Contains(detail, "release candidate is pending") || strings.HasPrefix(detail, "conflict:") {
			t.Fatalf("got %q %q", code, detail)
		}
	})

	t.Run("approving a draft is 409 with the capitalized reason", func(t *testing.T) {
		draftID, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Draft", "Draft change", editorID)
		mustNoErr("register draft", err)
		rec := doJSON(http.MethodPost, "/api/sops/"+sopID+"/versions/"+draftID+"/approve", approverToken, nil)
		want(t, rec, http.StatusConflict)
		if body := strings.TrimSpace(rec.Body.String()); body != "Only release candidates can be approved, current state is 'draft'" {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("reader acknowledgment of an unknown version is 404", func(t *testing.T) {
		rec := doJSON(http.MethodPost, "/api/sops/"+sopID+"/versions/no-such-version/add-reader", readerToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("duplicate reader acknowledgment is 409 already acknowledged", func(t *testing.T) {
		path := "/api/sops/" + sopID + "/versions/" + publishedID + "/add-reader"
		want(t, doJSON(http.MethodPost, path, readerToken, nil), http.StatusCreated)
		rec := doJSON(http.MethodPost, path, readerToken, nil)
		want(t, rec, http.StatusConflict)
		if body := strings.TrimSpace(rec.Body.String()); body != "already acknowledged" {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("diff against an unknown version is 404", func(t *testing.T) {
		rec := doJSON(http.MethodGet, "/api/sops/"+sopID+"/versions/"+publishedID+"/diff?against=no-such-version", readerToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("diff against another SOP's version is 404", func(t *testing.T) {
		rec := doJSON(http.MethodGet, "/api/sops/"+sopID+"/versions/"+publishedID+"/diff?against="+otherVersionID, readerToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("diff of a version under the wrong SOP is 404", func(t *testing.T) {
		rec := doJSON(http.MethodGet, "/api/sops/"+sopID+"/versions/"+otherVersionID+"/diff", readerToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("signature coverage for an unknown SOP is 404", func(t *testing.T) {
		rec := doJSON(http.MethodGet, "/api/sops/no-such-sop/training", approverToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("favoriting an unknown SOP is 404", func(t *testing.T) {
		rec := doJSON(http.MethodPost, "/api/sops/no-such-sop/favorite", readerToken, nil)
		want(t, rec, http.StatusNotFound)
	})

	t.Run("uploading an asset name twice is 409 asset_exists naming the file", func(t *testing.T) {
		path := "/api/sops/" + sopID + "/assets"
		want(t, upload(path, editorToken, "diagram.png"), http.StatusCreated)
		rec := upload(path, editorToken, "diagram.png")
		want(t, rec, http.StatusConflict)
		code, detail := jsonError(t, rec)
		if code != "asset_exists" || !strings.Contains(detail, `A file named "diagram.png" already exists`) {
			t.Fatalf("got %q %q", code, detail)
		}
	})
}

// TestAPI_UploadAndIntegrationErrorStatuses pins the statuses and JSON codes
// for avatar, logo and integration failures that come from error kinds.
func TestAPI_UploadAndIntegrationErrorStatuses(t *testing.T) {
	env := testenv.New(t)
	adminID := "kinds-upload-admin"
	env.SeedTestUser(t, adminID, "upload-admin@kinds.local", auth.RoleAdmin)
	token, _ := auth.GenerateAccessToken(adminID, auth.RoleAdmin)

	var ip int
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		ip++
		req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", ip%250+1)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}
	upload := func(path string, content []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("file", "picture.png")
		_, _ = part.Write(content)
		_ = w.Close()
		req := httptest.NewRequest(http.MethodPost, path, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		return serve(req)
	}
	wantJSON := func(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != status || body["error"] != code {
			t.Fatalf("got %d %q, want %d %q; body %q", rec.Code, body["error"], status, code, rec.Body.String())
		}
	}

	t.Run("avatar that isn't an image is 400 invalid_image", func(t *testing.T) {
		wantJSON(t, upload("/api/auth/me/avatar", []byte("not an image")), http.StatusBadRequest, "invalid_image")
	})
	t.Run("avatar over 8192 pixels is 400 image_too_large", func(t *testing.T) {
		wantJSON(t, upload("/api/auth/me/avatar", makeTestPNG(t, 8193, 1)), http.StatusBadRequest, "image_too_large")
	})
	t.Run("removing a missing avatar is 404 no_avatar", func(t *testing.T) {
		rec := serve(httptest.NewRequest(http.MethodDelete, "/api/auth/me/avatar", nil))
		wantJSON(t, rec, http.StatusNotFound, "no_avatar")
	})
	t.Run("an unknown avatar size is 400 invalid_size", func(t *testing.T) {
		rec := serve(httptest.NewRequest(http.MethodGet, "/api/users/"+adminID+"/avatar?size=huge", nil))
		wantJSON(t, rec, http.StatusBadRequest, "invalid_size")
	})
	t.Run("logo that isn't an image is 400 invalid_image", func(t *testing.T) {
		wantJSON(t, upload("/api/admin/settings/branding/logo", []byte("<svg/>")), http.StatusBadRequest, "invalid_image")
	})
	t.Run("logo over 4096 pixels is 400 image_too_large", func(t *testing.T) {
		wantJSON(t, upload("/api/admin/settings/branding/logo", makeTestPNG(t, 4097, 1)), http.StatusBadRequest, "image_too_large")
	})
	t.Run("integration with a malformed URL is 400 with the URL message", func(t *testing.T) {
		b, _ := json.Marshal(map[string]any{"enabled": true, "url": "https://", "events": []string{}})
		req := httptest.NewRequest(http.MethodPut, "/api/admin/settings/integrations/webhook", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := serve(req)
		if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != "URL must be a valid http(s) URL" {
			t.Fatalf("got %d %q", rec.Code, rec.Body.String())
		}
	})
}
