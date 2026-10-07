package audit_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	_ "modernc.org/sqlite"
)

// setupFileDB opens a WAL-mode SQLite file like production does, so several
// pooled connections share one database. The in-memory helper cannot show
// races between connections.
func setupFileDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		t.Fatalf("enable WAL: %v", err)
	}
	if _, err := db.Exec(`
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
	`); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	return db
}

// Concurrent writes without a transaction (as the notification goroutines,
// mail service, startup log, and PDF artifact log do) must still form one
// unbroken chain. Writers pass either nil or the pool itself.
func TestLog_ConcurrentStandaloneWritesKeepChainIntact(t *testing.T) {
	t.Setenv("AUDIT_BUSY_TIMEOUT", "5s")
	db := setupFileDB(t)
	loggerA, err := audit.New(db)
	if err != nil {
		t.Fatalf("new first logger: %v", err)
	}
	loggerB, err := audit.New(db)
	if err != nil {
		t.Fatalf("new second logger: %v", err)
	}

	const writers = 50
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	start := make(chan struct{})

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			var executor audit.DBTX
			if i%2 == 0 {
				executor = db
			}
			logger := loggerA
			if i%3 == 0 {
				logger = loggerB
			}
			errs <- logger.Log(executor, "notification_sent", "notification", fmt.Sprintf("channel-%d", i), nil, map[string]int{"n": i})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Log returned error: %v", err)
		}
	}

	report, err := loggerA.VerifyEntireChain()
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if report.Checked != writers {
		t.Errorf("checked %d events, want %d", report.Checked, writers)
	}
	if report.Invalid != 0 || len(report.Failed) != 0 {
		t.Errorf("chain has %d invalid events: %+v", report.Invalid, report.Failed)
	}
}

func TestLog_StandaloneWriteSurfacesBusyTimeout(t *testing.T) {
	t.Setenv("AUDIT_BUSY_TIMEOUT", "20ms")
	db := setupFileDB(t)
	logger, err := audit.New(db)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	blocker, err := db.Conn(t.Context())
	if err != nil {
		t.Fatalf("reserve blocker connection: %v", err)
	}
	defer blocker.Close()
	if _, err := blocker.ExecContext(t.Context(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("begin blocker transaction: %v", err)
	}

	err = logger.Log(nil, "notification_sent", "notification", "blocked", nil, map[string]bool{"blocked": true})
	if err == nil {
		t.Fatal("Log returned nil while another writer held the lock past the configured timeout")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events").Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("found %d audit rows after timed-out write, want 0", count)
	}

	if _, err := blocker.ExecContext(t.Context(), "ROLLBACK"); err != nil {
		t.Fatalf("release blocker transaction: %v", err)
	}
	if err := logger.Log(nil, "notification_sent", "notification", "recovered", nil, map[string]bool{"blocked": false}); err != nil {
		t.Fatalf("Log did not recover after lock release: %v", err)
	}
}

func TestNew_RejectsInvalidBusyTimeout(t *testing.T) {
	t.Setenv("AUDIT_BUSY_TIMEOUT", "not-a-duration")
	db := setupFileDB(t)
	if _, err := audit.New(db); err == nil {
		t.Fatal("New accepted an invalid AUDIT_BUSY_TIMEOUT")
	}
}
