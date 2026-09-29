package audit

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Logger struct {
	db *sql.DB
}

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
	var exec DBTX = executor
	if exec == nil {
		exec = l.db
	}
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

	return insertEventRecord(exec, eventType, entityType, entityID, actorUserID, payloadJSON)
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
