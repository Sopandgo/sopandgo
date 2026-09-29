package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
)

// handleAddAsset: Uploads a binary file (multipart/form-data)
func (s *Server) handleAddAsset(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	// Limit upload size (e.g., 20MB)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "multipart_invalid",
			"File too large (over 20 MB) or the upload was interrupted. Try a smaller file or retry.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "missing_file", "No file was sent. Choose a file and try again.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "read_failed", "Could not read the uploaded file. Please try again.")
		return
	}

	actorID := GetUserID(r.Context())

	assetID, err := s.sopService.AddAsset(sopID, header.Filename, content, &actorID)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.HasPrefix(msg, "asset already exists:"):
			fn := strings.TrimSpace(strings.TrimPrefix(msg, "asset already exists:"))
			writeJSONError(w, http.StatusConflict, "asset_exists",
				`A file named "`+fn+`" already exists for this SOP. Rename the file or remove the existing asset first.`)
		case strings.Contains(msg, "invalid path construction"):
			writeJSONError(w, http.StatusBadRequest, "invalid_asset_filename",
				"That file name is not allowed. Use a simple file name without folders or \"..\" (e.g. diagram.png).")
		default:
			writeJSONError(w, http.StatusInternalServerError, "upload_failed", "Could not save the file. Please try again.")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": assetID})
}

// handleGetAsset returns the metadata (path, hash, created_at) but NOT the content
func (s *Server) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	assetID := r.PathValue("assetID")
	if assetID == "" {
		http.Error(w, "missing assetID", http.StatusBadRequest)
		return
	}

	// Call service layer to get metadata
	asset, err := s.sopService.GetAssetByID(assetID)
	if err != nil {
		// You might want to check if err is "sql.ErrNoRows" to return 404
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// handleListAssets: Returns JSON list of assets for a SOP
func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	assets, err := s.sopService.GetAssetsBySOPID(sopID)
	if err != nil {
		http.Error(w, "Failed to list assets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// handleDownloadAsset: Streams the file content.
func (s *Server) handleDownloadAsset(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	assetID := r.PathValue("assetID")
	if assetID == "" {
		http.Error(w, "missing assetID", http.StatusBadRequest)
		return
	}

	absPath, err := s.sopService.GetAssetPath(assetID)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, absPath)
}

// handleCheckAssetIntegrity: Verify file hash matches DB record
func (s *Server) handleCheckAssetIntegrity(w http.ResponseWriter, r *http.Request) {
	sopID := r.PathValue("sopID")
	if sopID == "" {
		http.Error(w, "missing sopID", http.StatusBadRequest)
		return
	}

	assetID := r.PathValue("assetID")
	if assetID == "" {
		http.Error(w, "missing assetID", http.StatusBadRequest)
		return
	}

	valid, err := s.sopService.VerifyAssetIntegrity(assetID)
	if err != nil {
		// If file missing or DB error
		if os.IsNotExist(err) {
			http.Error(w, "File missing on disk", http.StatusNotFound)
			return
		}
		http.Error(w, "Integrity check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"hash_valid": valid})
}
