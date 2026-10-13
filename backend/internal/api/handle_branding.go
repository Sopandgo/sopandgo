package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/sopandgo/sopandgo/backend/internal/branding"
)

// handleGetBranding: GET /api/branding (public: the sign-in page needs it)
func (s *Server) handleGetBranding(w http.ResponseWriter, r *http.Request) {
	b, err := s.brandingService.Get()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "branding_unavailable", "Could not load branding.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// handleGetBrandingLogo: GET /api/branding/logo?variant=logo|icon (public)
func (s *Server) handleGetBrandingLogo(w http.ResponseWriter, r *http.Request) {
	variant := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("variant")))
	if variant == "" {
		variant = branding.VariantLogo
	}
	data, hash, err := s.brandingService.Image(variant)
	switch {
	case errors.Is(err, branding.ErrInvalidVariant):
		writeJSONError(w, http.StatusBadRequest, "invalid_variant", "variant must be logo or icon.")
		return
	case errors.Is(err, branding.ErrNoLogo):
		http.Error(w, "Logo not found", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, "Logo not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if hash != "" {
		etag := `"` + hash + `"`
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, _ = w.Write(data)
}

// handleAdminPutBranding: PUT /api/admin/settings/branding
func (s *Server) handleAdminPutBranding(w http.ResponseWriter, r *http.Request) {
	var in branding.TextInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&in); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	b, err := s.brandingService.UpdateText(in, GetUserID(r.Context()))
	switch {
	case errors.Is(err, branding.ErrNameInvalid):
		writeJSONError(w, http.StatusBadRequest, "name_invalid",
			"Use a name of at most 40 characters, on one line.")
		return
	case errors.Is(err, branding.ErrColourInvalid):
		writeJSONError(w, http.StatusBadRequest, "colour_invalid", "Use a colour like #1d4ed8.")
		return
	case errors.Is(err, branding.ErrColourContrast):
		writeJSONError(w, http.StatusBadRequest, "colour_contrast",
			"That colour is too close to the page background. Choose one with 4.5:1 contrast.")
		return
	case err != nil:
		writeJSONError(w, http.StatusInternalServerError, "save_failed", "Could not save branding. Please try again.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// handleAdminResetBranding: DELETE /api/admin/settings/branding
func (s *Server) handleAdminResetBranding(w http.ResponseWriter, r *http.Request) {
	b, err := s.brandingService.Reset(GetUserID(r.Context()))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "reset_failed", "Could not reset branding. Please try again.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// handleAdminUploadBrandingLogo: POST /api/admin/settings/branding/logo (multipart field "file")
func (s *Server) handleAdminUploadBrandingLogo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, branding.MaxUploadBytes+(64<<10))
	if err := r.ParseMultipartForm(branding.MaxUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "multipart_invalid",
			"File too large (over 2 MB) or the upload was interrupted. Try a smaller file or retry.")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing_file", "No file was sent. Choose a file and try again.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, branding.MaxUploadBytes+1))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "read_failed", "Could not read the uploaded file. Please try again.")
		return
	}
	if len(content) > branding.MaxUploadBytes {
		writeJSONError(w, http.StatusBadRequest, "file_too_large", "File too large (over 2 MB). Choose a smaller image.")
		return
	}

	b, err := s.brandingService.SetLogo(content, GetUserID(r.Context()))
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "unsupported image type"),
			strings.Contains(msg, "invalid image"),
			strings.Contains(msg, "empty image"):
			writeJSONError(w, http.StatusBadRequest, "invalid_image", "Use a PNG, JPEG, or WebP image.")
		case strings.Contains(msg, "image too large"):
			writeJSONError(w, http.StatusBadRequest, "image_too_large",
				"That image is too large. Use one under 4096 pixels on each side.")
		default:
			writeJSONError(w, http.StatusInternalServerError, "upload_failed", "Could not save the logo. Please try again.")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

// handleAdminDeleteBrandingLogo: DELETE /api/admin/settings/branding/logo
func (s *Server) handleAdminDeleteBrandingLogo(w http.ResponseWriter, r *http.Request) {
	b, err := s.brandingService.RemoveLogo(GetUserID(r.Context()))
	if errors.Is(err, branding.ErrNoLogo) {
		writeJSONError(w, http.StatusNotFound, "no_logo", "No logo to remove.")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "remove_failed", "Could not remove the logo. Please try again.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}
