package storage

import (
	"errors"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// IsUniqueViolation reports whether err, or an error it wraps, is SQLite
// refusing a row because a UNIQUE or PRIMARY KEY constraint already holds that
// value. It reads the driver's extended result code, never the message, so
// NOT NULL, CHECK and foreign key failures are not mistaken for duplicates.
func IsUniqueViolation(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	switch se.Code() {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
		return true
	default:
		return false
	}
}
