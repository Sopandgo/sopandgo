package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
)

const maxAvatarUploadBytes = 5 << 20

// handleUploadMyAvatar: POST /api/auth/me/avatar (multipart field "file")
func (s *Server) handleUploadMyAvatar(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxAvatarUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "multipart_invalid",
			"File too large (over 5 MB) or the upload was interrupted. Try a smaller file or retry.")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing_file", "No file was sent. Choose a file and try again.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxAvatarUploadBytes+1))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "read_failed", "Could not read the uploaded file. Please try again.")
		return
	}
	if len(content) > maxAvatarUploadBytes {
		writeJSONError(w, http.StatusBadRequest, "file_too_large",
			"File too large (over 5 MB). Choose a smaller image.")
		return
	}

	userID := GetUserID(r.Context())
	if err := s.authService.SetAvatar(userID, content, &userID); err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "unsupported image type"),
			strings.Contains(msg, "invalid image"),
			strings.Contains(msg, "empty image"):
			writeJSONError(w, http.StatusBadRequest, "invalid_image",
				"Use a JPEG, PNG, or WebP image.")
		case strings.Contains(msg, "image too large"),
			strings.Contains(msg, "invalid image dimensions"):
			writeJSONError(w, http.StatusBadRequest, "image_too_large",
				"That image is too large. Use a photo under 8192 pixels on each side.")
		default:
			writeJSONError(w, http.StatusInternalServerError, "upload_failed", "Could not save the picture. Please try again.")
		}
		return
	}

	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "upload_failed", "Could not save the picture. Please try again.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

// handleDeleteMyAvatar: DELETE /api/auth/me/avatar
func (s *Server) handleDeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if err := s.authService.RemoveAvatar(userID, &userID); err != nil {
		if strings.Contains(err.Error(), "no avatar") {
			writeJSONError(w, http.StatusNotFound, "no_avatar", "No profile picture to remove.")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "remove_failed", "Could not remove the picture. Please try again.")
		return
	}

	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

// handleGetUserAvatar: GET /api/users/{userID}/avatar?size=sm|md|lg|xl
func (s *Server) handleGetUserAvatar(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID == "" {
		http.Error(w, "missing userID", http.StatusBadRequest)
		return
	}

	size := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("size")))
	if size == "" {
		size = "sm"
	}

	absPath, hash, err := s.authService.AvatarAbsolutePath(userID, size)
	if err != nil {
		if strings.Contains(err.Error(), "invalid size") {
			writeJSONError(w, http.StatusBadRequest, "invalid_size",
				"size must be sm, md, lg, or xl.")
			return
		}
		if os.IsNotExist(err) || strings.Contains(err.Error(), "no avatar") {
			http.Error(w, "Avatar not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if hash != "" {
		etag := `"` + hash + `"`
		w.Header().Set("ETag", etag)
		if match := r.Header.Get("If-None-Match"); match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	http.ServeFile(w, r, absPath)
}
