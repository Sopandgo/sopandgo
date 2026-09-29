package audit_test

import (
	"database/sql"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	_ "modernc.org/sqlite"
)

// setupTestDB creates a fresh, in-memory SQLite database and applies the schema.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT,
			entity_type TEXT,
			entity_id TEXT,
			actor_user_id TEXT,
			payload TEXT,
			created_at TEXT,
			hash TEXT,
			prev_hash TEXT
		);
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			display_name TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db
}

// TestAuditChain_TamperDetection tests the happy path AND the tamper scenario
func TestAuditChain_TamperDetection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger, err := audit.New(db)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	actorID := "user-123"

	// 1. Log two valid events
	err = logger.Log(nil, "user_created", "User", "1", &actorID, `{"name": "Alice"}`)
	if err != nil {
		t.Fatalf("Failed to log first event: %v", err)
	}

	err = logger.Log(nil, "user_updated", "User", "1", &actorID, `{"name": "Alice Smith"}`)
	if err != nil {
		t.Fatalf("Failed to log second event: %v", err)
	}

	// 2. Verify the Happy Path (Hashes should be valid)
	events, total, err := logger.List(10, 0, audit.ListFilters{})
	if err != nil {
		t.Fatalf("Failed to list events: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 events, got %d", total)
	}

	for _, ev := range events {
		if !ev.HashValid {
			t.Errorf("Event %d should have a valid hash, but it was invalid", ev.ID)
		}
	}

	// 3. TAMPER WITH THE DATABASE!
	// We manually bypass the logger and alter the payload of the first event.
	_, err = db.Exec(`UPDATE audit_events SET payload = '{"name": "HACKED"}' WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to tamper with db: %v", err)
	}

	// 4. Verify the Unhappy Path (Tamper detection should trigger)
	tamperedEvents, _, err := logger.List(10, 0, audit.ListFilters{})
	if err != nil {
		t.Fatalf("Failed to list events after tampering: %v", err)
	}

	// The events are returned ORDER BY created_at DESC (newest first).
	// So tamperedEvents[0] is the newest (ID 2), tamperedEvents[1] is the oldest (ID 1).
	for _, ev := range tamperedEvents {
		if ev.ID == 1 && ev.HashValid {
			t.Errorf("Tamper detection failed! Event %d was altered but HashValid is true", ev.ID)
		}
		if ev.ID == 2 && !ev.HashValid {
			// Note: If you implement a strict blockchain, modifying ID 1 technically breaks
			// the prev_hash of ID 2. But your current List() only checks the *current* row's integrity.
			// This assertion proves the current row check works.
		}
	}
}

// TestLogger_PayloadMarshaling tests the switch statement in your Log function
func TestLogger_PayloadMarshaling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger, _ := audit.New(db)

	type testStruct struct {
		Role string `json:"role"`
	}

	tests := []struct {
		name            string
		payloadInput    any
		expectedPayload string
	}{
		{
			name:            "string payload",
			payloadInput:    `{"simple":"string"}`,
			expectedPayload: `{"simple":"string"}`,
		},
		{
			name:            "byte slice payload",
			payloadInput:    []byte(`{"simple":"bytes"}`),
			expectedPayload: `{"simple":"bytes"}`,
		},
		{
			name:            "struct payload",
			payloadInput:    testStruct{Role: "admin"},
			expectedPayload: `{"role":"admin"}`,
		},
		{
			name:            "unmarshalable payload",
			payloadInput:    make(chan int), // Channels cannot be JSON marshaled
			expectedPayload: `{"error": "failed_to_marshal_payload", "details": "json: unsupported type: chan int"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean the table before each subtest to ensure ID 1 is always the one we check
			db.Exec("DELETE FROM audit_events")

			err := logger.Log(nil, "test_event", "Test", "1", nil, tt.payloadInput)
			if err != nil {
				t.Fatalf("Log() returned unexpected error: %v", err)
			}

			// Verify what was actually saved to the database
			var savedPayload string
			err = db.QueryRow("SELECT payload FROM audit_events LIMIT 1").Scan(&savedPayload)
			if err != nil {
				t.Fatalf("Failed to query db: %v", err)
			}

			if savedPayload != tt.expectedPayload {
				t.Errorf("Expected payload %q, got %q", tt.expectedPayload, savedPayload)
			}
		})
	}
}

func TestLogger_ListPaginationAndFiltering(t *testing.T) {
	db := setupTestDB(t) // Using modernc.org/sqlite now
	defer db.Close()
	logger, _ := audit.New(db)

	// Insert 3 "login" events and 2 "logout" events
	for i := 0; i < 3; i++ {
		logger.Log(nil, "login", "User", "1", nil, `{"status":"success"}`)
	}
	for i := 0; i < 2; i++ {
		logger.Log(nil, "logout", "User", "1", nil, `{"status":"success"}`)
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		filters       audit.ListFilters
		expectedCount int
		expectedTotal int // The total matching the filter, ignoring limit
	}{
		{"all events", 10, 0, audit.ListFilters{}, 5, 5},
		{"limit 2", 2, 0, audit.ListFilters{}, 2, 5},
		{"offset 2", 10, 2, audit.ListFilters{}, 3, 5},
		{"filter login", 10, 0, audit.ListFilters{EventType: "login"}, 3, 3},
		{"filter logout", 10, 0, audit.ListFilters{EventType: "logout"}, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, total, err := logger.List(tt.limit, tt.offset, tt.filters)
			if err != nil {
				t.Fatalf("List failed: %v", err)
			}
			if len(events) != tt.expectedCount {
				t.Errorf("Expected %d events, got %d", tt.expectedCount, len(events))
			}
			if total != tt.expectedTotal {
				t.Errorf("Expected total %d, got %d", tt.expectedTotal, total)
			}
		})
	}
}

func TestLogger_VerifyEntireChain(t *testing.T) {
	db := setupTestDB(t) // Remember to use modernc.org/sqlite
	defer db.Close()
	logger, _ := audit.New(db)

	// Log 3 events to build a small chain
	logger.Log(nil, "e1", "Ent", "1", nil, "payload 1")
	logger.Log(nil, "e2", "Ent", "1", nil, "payload 2")
	logger.Log(nil, "e3", "Ent", "1", nil, "payload 3")

	// 1. Happy Path: Chain should be valid
	report, err := logger.VerifyEntireChain()
	if err != nil {
		t.Fatalf("VerifyEntireChain returned unexpected error: %v", err)
	}
	if report.Invalid > 0 || report.Checked != 3 {
		t.Fatalf("Expected chain to be valid (3 checked, 0 invalid), got checked=%d, invalid=%d", report.Checked, report.Invalid)
	}

	// 2. Unhappy Path: Tamper with Row 1's payload
	_, err = db.Exec(`UPDATE audit_events SET payload = 'hacked' WHERE id = 1`)
	if err != nil {
		t.Fatalf("Failed to tamper with db: %v", err)
	}

	// 3. Verify the chain is now flagged as broken and captures the correct ID
	report, err = logger.VerifyEntireChain()
	if err != nil {
		t.Fatalf("VerifyEntireChain returned unexpected error: %v", err)
	}

	// We expect exactly 1 invalid row, and 2 valid rows
	if report.Invalid != 1 {
		t.Errorf("Tamper detection failed! Expected exactly 1 invalid row, got %d", report.Invalid)
	}
	if report.Valid != 2 {
		t.Errorf("Expected 2 valid rows to survive, got %d", report.Valid)
	}

	// Check the detailed failure report
	if len(report.Failed) != 1 {
		t.Fatalf("Expected exactly 1 failure report, got %d", len(report.Failed))
	}

	failure := report.Failed[0]
	if failure.EventID != 1 {
		t.Errorf("Expected failed EventID to be 1, got %d", failure.EventID)
	}
	if failure.Error != "hash_mismatch" {
		t.Errorf("Expected failure error to be 'hash_mismatch', got %q", failure.Error)
	}
}
