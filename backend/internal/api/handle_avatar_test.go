package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func TestAPI_Avatar(t *testing.T) {
	env := testenv.New(t)

	userID := "avatar-user-1"
	otherID := "avatar-user-2"
	env.SeedTestUser(t, userID, "avatar@api.local", auth.RoleEditor)
	env.SeedTestUser(t, otherID, "other@api.local", auth.RoleViewer)
	if err := env.AuthService.UpdateUserPassword(userID, "SuperSecret123!", &userID); err != nil {
		t.Fatalf("password: %v", err)
	}

	token, err := auth.GenerateAccessToken(userID, auth.RoleEditor)
	if err != nil {
		t.Fatal(err)
	}
	otherToken, err := auth.GenerateAccessToken(otherID, auth.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}

	do := func(method, path, tok string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body != nil {
			reader = bytes.NewReader(body.Bytes())
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Authorization", "Bearer "+tok)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		return rec
	}

	t.Run("GET without avatar is 404", func(t *testing.T) {
		rec := do(http.MethodGet, "/api/users/"+userID+"/avatar?size=sm", token, nil, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("reject non-image", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		part, _ := w.CreateFormFile("file", "note.txt")
		_, _ = part.Write([]byte("not an image"))
		_ = w.Close()
		rec := do(http.MethodPost, "/api/auth/me/avatar", token, body, w.FormDataContentType())
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
	})

	var contentHash string
	t.Run("upload PNG and list sizes", func(t *testing.T) {
		pngBytes := makeTestPNG(t, 2000, 1800)
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		part, _ := w.CreateFormFile("file", "photo.png")
		_, _ = part.Write(pngBytes)
		_ = w.Close()

		rec := do(http.MethodPost, "/api/auth/me/avatar", token, body, w.FormDataContentType())
		if rec.Code != http.StatusOK {
			t.Fatalf("upload %d %s", rec.Code, rec.Body.String())
		}
		var user auth.User
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatal(err)
		}
		if !user.HasAvatar || user.AvatarContentHash == "" {
			t.Fatalf("expected has_avatar and hash, got %+v", user)
		}
		contentHash = user.AvatarContentHash

		for _, px := range []int{96, 256, 512, 1024} {
			path := filepath.Join(env.DataDir, "users", userID, fmt.Sprintf("avatar-%d.jpg", px))
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("missing %s: %v", path, err)
			}
			cfg, err := jpeg.DecodeConfig(f)
			f.Close()
			if err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
			if cfg.Width != px || cfg.Height != px {
				t.Fatalf("%s: got %dx%d", path, cfg.Width, cfg.Height)
			}
		}
		// Small list size should be much smaller than a large source photo.
		smInfo, err := os.Stat(filepath.Join(env.DataDir, "users", userID, "avatar-96.jpg"))
		if err != nil {
			t.Fatal(err)
		}
		if smInfo.Size() >= int64(len(pngBytes))/4 {
			t.Fatalf("avatar-96.jpg unexpectedly large: %d vs source %d", smInfo.Size(), len(pngBytes))
		}
	})

	t.Run("GET sizes and ETag", func(t *testing.T) {
		for _, size := range []string{"sm", "md", "lg", "xl"} {
			rec := do(http.MethodGet, "/api/users/"+userID+"/avatar?size="+size, otherToken, nil, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("size %s: %d %s", size, rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "image/jpeg" {
				t.Fatalf("content-type %q", ct)
			}
			etag := rec.Header().Get("ETag")
			want := `"` + contentHash + `"`
			if etag != want {
				t.Fatalf("etag %q want %q", etag, want)
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/avatar?size=sm", nil)
		req.Header.Set("Authorization", "Bearer "+otherToken)
		req.Header.Set("If-None-Match", `"`+contentHash+`"`)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotModified {
			t.Fatalf("expected 304, got %d", rec.Code)
		}
	})

	t.Run("me has_avatar", func(t *testing.T) {
		rec := do(http.MethodGet, "/api/auth/me", token, nil, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("me %d", rec.Code)
		}
		var user auth.User
		_ = json.Unmarshal(rec.Body.Bytes(), &user)
		if !user.HasAvatar {
			t.Fatal("expected has_avatar")
		}
	})

	t.Run("invalid size", func(t *testing.T) {
		rec := do(http.MethodGet, "/api/users/"+userID+"/avatar?size=huge", token, nil, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/"+userID+"/avatar", nil)
		rec := httptest.NewRecorder()
		env.API.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("delete", func(t *testing.T) {
		rec := do(http.MethodDelete, "/api/auth/me/avatar", token, nil, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("delete %d %s", rec.Code, rec.Body.String())
		}
		var user auth.User
		_ = json.Unmarshal(rec.Body.Bytes(), &user)
		if user.HasAvatar {
			t.Fatal("expected has_avatar false")
		}
		rec = do(http.MethodGet, "/api/users/"+userID+"/avatar?size=sm", token, nil, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", rec.Code)
		}
	})
}
