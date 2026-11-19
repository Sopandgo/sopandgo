package audit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Logger struct {
	db *sql.DB
}

// New returns a logger that writes to db. The chain relies on db being opened
// by the storage package, whose transactions hold the write lock from begin.
func New(db *sql.DB) (*Logger, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to audit database: %w", err)
	}
	strictAuditTypes = strings.EqualFold(os.Getenv("AUDIT_STRICT_TYPES"), "true")
	return &Logger{db: db}, nil
}

// Log writes an audit entry.
// payload can be a string (assumed to be pre-encoded JSON) or any struct/map (which will be Marshaled).
func (l *Logger) Log(executor DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error {
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	if strictAuditTypes {
		if _, ok := knownEventTypes[eventType]; !ok {
			return fmt.Errorf("unknown audit event_type: %q", eventType)
		}
		if _, ok := knownEntityTypes[entityType]; !ok {
			return fmt.Errorf("unknown audit entity_type: %q", entityType)
		}
	}

	var payloadJSON string

	switch v := payload.(type) {
	case string:
		// Trust the caller provided valid JSON string
		payloadJSON = v
	case []byte:
		payloadJSON = string(v)
	default:
		// Automatically Marshal maps and structs
		b, err := json.Marshal(v)
		if err != nil {
			//Log a "corruption" error payload so the audit trail remains intact.
			payloadJSON = fmt.Sprintf(`{"error": "failed_to_marshal_payload", "details": "%s"}`, err.Error())
		} else {
			payloadJSON = string(b)
		}
	}

	if executor == nil {
		return l.insertStandalone(l.db, eventType, entityType, entityID, actorUserID, payloadJSON)
	}
	if db, ok := executor.(*sql.DB); ok {
		return l.insertStandalone(db, eventType, entityType, entityID, actorUserID, payloadJSON)
	}
	return insertEventRecord(executor, eventType, entityType, entityID, actorUserID, payloadJSON)
}

// insertStandalone appends an event when the caller has no transaction.
//
// Reading the chain tip and inserting the new row must happen under one write
// lock. As two autocommit statements on a pooled connection, two concurrent
// callers (for example the parallel notification goroutines) can read the same
// tip and both insert a row pointing at it, which forks the chain and makes
// VerifyEntireChain report a broken link that is not tampering.
//
// Every transaction holds SQLite's write lock from begin (the storage package
// opens connections with immediate transactions and a busy timeout), so the
// next writer waits and then reads the tip this call wrote. A writer that
// cannot get the lock within the busy timeout gets the error back.
func (l *Logger) insertStandalone(db *sql.DB, eventType, entityType, entityID string, actorUserID *string, payloadJSON string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertEventRecord(tx, eventType, entityType, entityID, actorUserID, payloadJSON); err != nil {
		return err
	}
	return tx.Commit()
}

// List retrieves audit events, total count, and performs integrity checks.
func (l *Logger) List(limit, offset int, filters ListFilters) ([]AuditEventWithVerification, int, error) {
	// 1. Fetch raw records AND total count
	rawEvents, total, err := listEventsRecord(l.db, limit, offset, filters)
	if err != nil {
		return nil, 0, err
	}

	verifiedEvents := make([]AuditEventWithVerification, len(rawEvents))

	for i, raw := range rawEvents {
		verifiedEvents[i] = AuditEventWithVerification{
			AuditEvent: raw,
			HashValid:  raw.rowHashValid(),
		}
	}

	return verifiedEvents, total, nil
}

func (l *Logger) ListFilterOptions() (FilterOptions, error) {
	return listDistinctAuditTypesRecord(l.db)
}

// VerifyEntireChain reads the audit log sequentially to ensure no links are broken.
// It returns a detailed report of the chain's health.
func (l *Logger) VerifyEntireChain() (ChainIntegrityReport, error) {
	events, err := listEventsAscRecord(l.db, 0)
	if err != nil {
		return ChainIntegrityReport{}, err
	}
	return verifyEvents(events, GenesisHash), nil
}

// ErrAnchorMoved means the row an anchor points at no longer has the hash that
// was anchored, or no longer exists.
var ErrAnchorMoved = errors.New("audit: anchored row changed or missing")

// VerifySince checks the chain after an anchored row: that row id still has
// hash, every later row links to the one before it, and every later row hash
// recomputes. afterID 0 with GenesisHash checks the whole chain.
func (l *Logger) VerifySince(afterID int64, hash string) (ChainIntegrityReport, error) {
	if afterID > 0 {
		stored, err := eventHashRecord(l.db, afterID)
		if errors.Is(err, sql.ErrNoRows) {
			return ChainIntegrityReport{}, ErrAnchorMoved
		}
		if err != nil {
			return ChainIntegrityReport{}, err
		}
		if stored != hash {
			return ChainIntegrityReport{}, ErrAnchorMoved
		}
	}
	events, err := listEventsAscRecord(l.db, afterID)
	if err != nil {
		return ChainIntegrityReport{}, err
	}
	report := verifyEvents(events, hash)
	if report.Checked == 0 {
		report.TipID, report.TipHash = afterID, hash
	}
	return report, nil
}

// Tip returns the newest event's id and hash, or (0, GenesisHash) when the
// log is empty.
func (l *Logger) Tip() (int64, string, error) {
	return tipRecord(l.db)
}

// ChainEvents returns every event in chain order, for verification and export.
func (l *Logger) ChainEvents() ([]AuditEvent, error) {
	return listEventsAscRecord(l.db, 0)
}

// verifyEvents walks events in id order starting from expectedPrevHash.
func verifyEvents(events []AuditEvent, expectedPrevHash string) ChainIntegrityReport {
	report := ChainIntegrityReport{TipHash: expectedPrevHash}

	for _, ev := range events {
		report.Checked++
		var errMsg string

		// A. Check if the chain link is broken, then B. whether the row's own
		// hash was tampered with.
		if ev.PrevHash != expectedPrevHash {
			errMsg = "broken_chain_link"
		} else if !ev.rowHashValid() {
			errMsg = "hash_mismatch"
		}

		// C. Tally the results
		if errMsg == "" {
			report.Valid++
		} else {
			report.Invalid++
			report.Failed = append(report.Failed, ChainFailure{
				EventID: ev.ID,
				Error:   errMsg,
			})
		}

		// D. Resync: the current row's stored hash becomes the expected
		// prev_hash for the next row even if this row was tampered with, so one
		// edit does not cascade into failures for every later row.
		expectedPrevHash = ev.Hash
		report.TipID, report.TipHash = ev.ID, ev.Hash
	}

	return report
}
