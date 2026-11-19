package api

import (
	"net/http"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

// kindStatus maps an error's kind to its HTTP status. This is the only place
// that decides 409, 404 or 400 for a domain error. ok is false when err has no
// kind; the handler then answers its own 500. Handlers still choose the body.
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

// writeKindJSON answers err as a JSON error with the handler's code and detail,
// at the status err's kind maps to. It writes nothing and reports false when
// err has no kind.
func writeKindJSON(w http.ResponseWriter, err error, code, detail string) bool {
	status, ok := kindStatus(err)
	if !ok {
		return false
	}
	writeJSONError(w, status, code, detail)
	return true
}

// writeKindText is writeKindJSON for handlers that answer in plain text.
func writeKindText(w http.ResponseWriter, err error, body string) bool {
	status, ok := kindStatus(err)
	if !ok {
		return false
	}
	http.Error(w, body, status)
	return true
}
