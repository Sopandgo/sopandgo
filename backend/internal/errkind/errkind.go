// Package errkind marks an error with what kind of failure it is: a conflict
// with the current state, a missing record, or an invalid request.
//
// Domain modules mark the errors a client can cause, and the HTTP module maps
// the kind to a status code in one place. The message stays free text for
// people to read and never decides the status.
package errkind

import (
	"errors"
	"fmt"
)

// Kind is one kind of failure. Use it as the target of errors.Is.
type Kind int

const (
	// Conflict: the request clashes with the current state (HTTP 409).
	Conflict Kind = iota + 1
	// NotFound: the record the request names does not exist (HTTP 404).
	NotFound
	// Invalid: the request itself is malformed or not allowed (HTTP 400).
	Invalid
)

var kinds = []Kind{Conflict, NotFound, Invalid}

func (k Kind) Error() string {
	switch k {
	case Conflict:
		return "conflict"
	case NotFound:
		return "not found"
	case Invalid:
		return "invalid"
	default:
		return "unknown error kind"
	}
}

type kindError struct {
	kind Kind
	err  error
}

func (e *kindError) Error() string        { return e.err.Error() }
func (e *kindError) Unwrap() error        { return e.err }
func (e *kindError) Is(target error) bool { return target == e.kind }

// New returns an error of the given kind. The text is the formatted message
// alone. A %w verb in format wraps a cause, which errors.Is and errors.As still
// reach. Each call returns a distinct error, so a package-level New works as a
// named error that is also of its kind.
func New(kind Kind, format string, args ...any) error {
	return &kindError{kind: kind, err: fmt.Errorf(format, args...)}
}

type detailError struct {
	named error
	err   error
}

func (e *detailError) Error() string   { return e.err.Error() }
func (e *detailError) Unwrap() []error { return []error{e.named, e.err} }

// Detail returns an instance of a named error made with New: errors.Is matches
// named and its kind, but the text is this call's formatted message alone. Use
// it when one named error is reported with different messages. A %w verb in
// format wraps a cause.
func Detail(named error, format string, args ...any) error {
	return &detailError{named: named, err: fmt.Errorf(format, args...)}
}

// KindOf reports the kind of err, looking through wrapping. ok is false when
// err is nil or not marked with a kind, whatever its text says.
func KindOf(err error) (kind Kind, ok bool) {
	for _, k := range kinds {
		if errors.Is(err, k) {
			return k, true
		}
	}
	return 0, false
}
