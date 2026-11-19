package api

import (
	"log"
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

// kindStatus maps an error's kind to its HTTP status: conflict 409, not found
// 404, invalid 400. ok is false when err has no kind. Handlers still choose the
// body. Named errors without a kind (backup, mail, two-factor, …) keep their
// own explicit statuses in their handlers.
func kindStatus(err error) (status int, ok bool) {
	kind, ok := errkind.KindOf(err)
	if !ok {
		return 0, false
	}
	switch kind {
	case errkind.Conflict:
		return http.StatusConflict, true
	case errkind.NotFound:
		return http.StatusNotFound, true
	case errkind.Invalid:
		return http.StatusBadRequest, true
	default:
		return 0, false
	}
}

// statusOrInternal is kindStatus with a 500 for an error that has no kind, so
// a handler that expected a kind always writes a response.
func statusOrInternal(err error) int {
	if status, ok := kindStatus(err); ok {
		return status
	}
	log.Printf("ERROR: expected an error kind, got %v", err)
	return http.StatusInternalServerError
}

// writeKindJSON answers err as a JSON error with the handler's code and detail,
// at the status err's kind maps to.
func writeKindJSON(w http.ResponseWriter, err error, code, detail string) {
	writeJSONError(w, statusOrInternal(err), code, detail)
}

// writeKindText is writeKindJSON for handlers that answer in plain text.
func writeKindText(w http.ResponseWriter, err error, body string) {
	http.Error(w, body, statusOrInternal(err))
}
