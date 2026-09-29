package audit

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"
)

// GenesisHash is the "parent" of the very first log entry.
// It is an arbitrary constant (SHA256 of 64 zeros) used to anchor the chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func insertEventRecord(db DBTX, eventType, entityType, entityID string, actorUserID *string, payloadJSON string) error {
	var prevHash string

	// 1. Get the Tip of the Chain using the Integer ID
	err := db.QueryRow(`
        SELECT hash FROM audit_events 
        ORDER BY id DESC 
        LIMIT 1
    `).Scan(&prevHash)

	if err != nil {
		prevHash = GenesisHash
	}

	// 2. Prepare Data
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	// 3. Calculate New Hash
	recordToHash := prevHash + eventType + entityID + payloadJSON + createdAt

	hasher := sha256.New()
	hasher.Write([]byte(recordToHash))
	currentHash := hex.EncodeToString(hasher.Sum(nil))

	// 4. Insert (SQLite provides the ID automatically)
	_, err = db.Exec(`
        INSERT INTO audit_events (
            event_type, entity_type, entity_id, actor_user_id, 
            payload, created_at, hash, prev_hash
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `,
		eventType,
		entityType,
		entityID,
		actorUserID,
		payloadJSON,
		createdAt,
		currentHash,
		prevHash,
	)

	return err
}

func listEventsRecord(db DBTX, limit, offset int, filters ListFilters) ([]AuditEvent, int, error) {
	whereClauses := make([]string, 0, 4)
	whereArgs := make([]interface{}, 0, 6)

	if filters.EventType != "" {
		whereClauses = append(whereClauses, "a.event_type = ?")
		whereArgs = append(whereArgs, filters.EventType)
	}
	if filters.EntityType != "" {
		whereClauses = append(whereClauses, "a.entity_type = ?")
		whereArgs = append(whereArgs, filters.EntityType)
	}
	if filters.ActorUserID != "" {
		whereClauses = append(whereClauses, "a.actor_user_id = ?")
		whereArgs = append(whereArgs, filters.ActorUserID)
	}
	if filters.SOPID != "" {
		whereClauses = append(whereClauses, "(a.entity_id = ? OR a.payload LIKE ? ESCAPE '\\')")
		whereArgs = append(whereArgs, filters.SOPID, `%`+escapeLike(filters.SOPID)+`%`)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// --- 1. Get Total Count ---
	// We count matching rows before applying Limit/Offset
	countQuery := `SELECT COUNT(*) FROM audit_events a` + whereSQL

	var total int
	// We use QueryRow because we expect exactly one result (the count)
	err := db.QueryRow(countQuery, whereArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// --- 2. Get Data ---
	query := `
        SELECT 
            a.id, a.event_type, a.entity_type, a.entity_id, 
            a.actor_user_id, u.display_name, a.payload, a.created_at,
            a.hash, a.prev_hash
        FROM audit_events a
        LEFT JOIN users u ON a.actor_user_id = u.id` + whereSQL

	query += " ORDER BY a.created_at DESC LIMIT ? OFFSET ?"
	args := append(whereArgs, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var createdAt string
		var actorName sql.NullString

		if err := rows.Scan(
			&e.ID, &e.EventType, &e.EntityType, &e.EntityID,
			&e.ActorID, &actorName, &e.Payload, &createdAt,
			&e.Hash, &e.PrevHash,
		); err != nil {
			return nil, 0, err
		}

		if actorName.Valid {
			e.ActorName = &actorName.String
		}

		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		events = append(events, e)
	}

	return events, total, nil
}

func escapeLike(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `%`, `\%`)
	v = strings.ReplaceAll(v, `_`, `\_`)
	return v
}

// listAllEventsAscRecord fetches all events in chronological order for chain verification.
func listAllEventsAscRecord(db DBTX) ([]AuditEvent, error) {
	query := `
		SELECT 
			id, event_type, entity_id, payload, created_at, hash, prev_hash 
		FROM audit_events 
		ORDER BY id ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var createdAt string

		if err := rows.Scan(
			&e.ID, &e.EventType, &e.EntityID, &e.Payload, &createdAt, &e.Hash, &e.PrevHash,
		); err != nil {
			return nil, err
		}

		e.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		events = append(events, e)
	}

	return events, nil
}

func listDistinctAuditTypesRecord(db DBTX) (FilterOptions, error) {
	readList := func(query string) ([]string, error) {
		rows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		out := make([]string, 0, 16)
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				return nil, err
			}
			if strings.TrimSpace(v) != "" {
				out = append(out, v)
			}
		}
		return out, nil
	}

	eventTypes, err := readList(`
		SELECT DISTINCT event_type
		FROM audit_events
		WHERE trim(event_type) <> ''
		ORDER BY event_type ASC
	`)
	if err != nil {
		return FilterOptions{}, err
	}

	entityTypes, err := readList(`
		SELECT DISTINCT entity_type
		FROM audit_events
		WHERE trim(entity_type) <> ''
		ORDER BY entity_type ASC
	`)
	if err != nil {
		return FilterOptions{}, err
	}

	return FilterOptions{
		EventTypes:  eventTypes,
		EntityTypes: entityTypes,
	}, nil
}
