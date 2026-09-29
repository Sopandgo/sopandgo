package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Explicitly set the status code
	w.WriteHeader(http.StatusOK)

	// Return an object, not just a string
	response := map[string]string{"status": "available"}

	json.NewEncoder(w).Encode(response)
}
