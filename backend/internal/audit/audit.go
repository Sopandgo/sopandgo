package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	db                    *sql.DB
	standaloneMu          sync.Mutex
	standaloneBusyTimeout time.Duration
}

const defaultStandaloneBusyTimeout = 5 * time.Second

func New(db *sql.DB) (*Logger, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to audit database: %w", err)
	}
	strictAuditTypes = strings.EqualFold(os.Getenv("AUDIT_STRICT_TYPES"), "true")
	busyTimeout := defaultStandaloneBusyTimeout
	if raw := strings.TrimSpace(os.Getenv("AUDIT_BUSY_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed < time.Millisecond {
			return nil, fmt.Errorf("AUDIT_BUSY_TIMEOUT must be a duration of at least 1ms")
		}
		busyTimeout = parsed
	}
	return &Logger{db: db, standaloneBusyTimeout: busyTimeout}, nil
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
// BEGIN IMMEDIATE takes SQLite's write lock before the tip is read, so the
// next writer waits and then reads the tip this call wrote.
func (l *Logger) insertStandalone(db *sql.DB, eventType, entityType, entityID string, actorUserID *string, payloadJSON string) error {
	// Avoid opening one waiting SQLite connection per concurrent goroutine.
	// The database lock below remains necessary when another Logger or process
	// writes the same database.
	l.standaloneMu.Lock()
	defer l.standaloneMu.Unlock()

	if db == nil {
		db = l.db
	}
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// busy_timeout is per connection; without it BEGIN IMMEDIATE fails at once
	// when another writer holds the lock.
	busyTimeoutMs := l.standaloneBusyTimeout.Milliseconds()
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout = %d", busyTimeoutMs)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK")
		}
	}()

	if err := insertEventRecord(connExecutor{ctx: ctx, conn: conn}, eventType, entityType, entityID, actorUserID, payloadJSON); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}

// connExecutor adapts a single *sql.Conn to DBTX.
type connExecutor struct {
	ctx  context.Context
	conn *sql.Conn
}

func (c connExecutor) Exec(query string, args ...any) (sql.Result, error) {
	return c.conn.ExecContext(c.ctx, query, args...)
}

func (c connExecutor) Query(query string, args ...any) (*sql.Rows, error) {
	return c.conn.QueryContext(c.ctx, query, args...)
}

func (c connExecutor) QueryRow(query string, args ...any) *sql.Row {
	return c.conn.QueryRowContext(c.ctx, query, args...)
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
		}

		// 2. Format the timestamp EXACTLY as it was stored
		createdAtStr := raw.CreatedAt.UTC().Format(time.RFC3339Nano)

		// 3. Reconstruct the hash recipe
		// Order must be: PrevHash + EventType + EntityID + Payload + CreatedAt
		recordToHash := raw.PrevHash +
			raw.EventType +
			raw.EntityID +
			raw.Payload +
			createdAtStr

		hasher := sha256.New()
		hasher.Write([]byte(recordToHash))
		calculatedHash := hex.EncodeToString(hasher.Sum(nil))

		// 4. Compare
		verifiedEvents[i].HashValid = (calculatedHash == raw.Hash)
	}

	return verifiedEvents, total, nil
}

func (l *Logger) ListFilterOptions() (FilterOptions, error) {
	return listDistinctAuditTypesRecord(l.db)
}

// audit.go

// VerifyEntireChain reads the audit log sequentially to ensure no links are broken.
// It returns a detailed report of the chain's health.
func (l *Logger) VerifyEntireChain() (ChainIntegrityReport, error) {
	events, err := listAllEventsAscRecord(l.db)
	if err != nil {
		return ChainIntegrityReport{}, err
	}

	report := ChainIntegrityReport{}
	expectedPrevHash := GenesisHash

	for _, ev := range events {
		report.Checked++
		rowValid := true
		var errMsg string

		// A. Check if the chain link is broken
		if ev.PrevHash != expectedPrevHash {
			rowValid = false
			errMsg = "broken_chain_link"
		} else {
			// B. Verify the row's internal hash hasn't been tampered with
			createdAtStr := ev.CreatedAt.UTC().Format(time.RFC3339Nano)
			recordToHash := ev.PrevHash + ev.EventType + ev.EntityID + ev.Payload + createdAtStr

			hasher := sha256.New()
			hasher.Write([]byte(recordToHash))
			calcHash := hex.EncodeToString(hasher.Sum(nil))

			if calcHash != ev.Hash {
				rowValid = false
				errMsg = "hash_mismatch"
			}
		}

		// C. Tally the results
		if rowValid {
			report.Valid++
		} else {
			report.Invalid++
			report.Failed = append(report.Failed, ChainFailure{
				EventID: ev.ID,
				Error:   errMsg,
			})
		}

		// D. Resync! The current row's stored hash becomes the expected prev_hash
		// for the next iteration, even if this row was tampered with. This prevents cascading failures.
		expectedPrevHash = ev.Hash
	}

	return report, nil
}
