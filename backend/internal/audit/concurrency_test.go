package audit_test

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// setupFileDB opens a file database through the storage package, so the tests
// run with the production connection settings and several pooled connections
// share one database. The in-memory helper cannot show races between
// connections.
func setupFileDB(t *testing.T, busyTimeout string) *sql.DB {
	t.Helper()
	t.Setenv("DB_BUSY_TIMEOUT", busyTimeout)
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })
	return store.DB
}

// The concurrency tests use 10 writers: more than production's worst case (a
// few notification channels per event), and enough to fail when transactions
// do not take the write lock at begin. Waiting writers poll in SQLite's busy
// handler, so far larger counts approach the busy timeout on a slow disk.
//
// runConcurrently starts n goroutines at once and returns their errors.
func runConcurrently(n int, fn func(i int) error) []error {
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
	return errs
}

func requireIntactChain(t *testing.T, logger *audit.Logger, want int) {
	t.Helper()
	report, err := logger.VerifyEntireChain()
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if report.Checked != want {
		t.Errorf("checked %d events, want %d", report.Checked, want)
	}
	if report.Invalid != 0 || len(report.Failed) != 0 {
		t.Errorf("chain has %d invalid events: %+v", report.Invalid, report.Failed)
	}
}

// Concurrent writes without a transaction (as the notification goroutines,
// mail service, startup log, and PDF artifact log do) must still form one
// unbroken chain. Writers pass either nil or the pool itself.
func TestLog_ConcurrentStandaloneWritesKeepChainIntact(t *testing.T) {
	db := setupFileDB(t, "5s")
	loggerA, err := audit.New(db)
	if err != nil {
		t.Fatalf("new first logger: %v", err)
	}
	loggerB, err := audit.New(db)
	if err != nil {
		t.Fatalf("new second logger: %v", err)
	}

	const writers = 10
	errs := runConcurrently(writers, func(i int) error {
		var executor audit.DBTX
		if i%2 == 0 {
			executor = db
		}
		logger := loggerA
		if i%3 == 0 {
			logger = loggerB
		}
		return logger.Log(executor, "notification_sent", "notification", fmt.Sprintf("channel-%d", i), nil, map[string]int{"n": i})
	})
	for _, err := range errs {
		if err != nil {
			t.Errorf("Log returned error: %v", err)
		}
	}

	requireIntactChain(t, loggerA, writers)
}

// A caller's transaction may log before it writes anything else. The chain
// must stay intact regardless of that ordering.
func TestLog_ConcurrentCallerTransactionsKeepChainIntact(t *testing.T) {
	db := setupFileDB(t, "5s")
	logger, err := audit.New(db)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	const writers = 10
	errs := runConcurrently(writers, func(i int) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err := logger.Log(tx, "sop_version_created", "sop_version", fmt.Sprintf("version-%d", i), nil, map[string]int{"n": i}); err != nil {
			return err
		}
		return tx.Commit()
	})
	for _, err := range errs {
		if err != nil {
			t.Errorf("transactional Log returned error: %v", err)
		}
	}

	requireIntactChain(t, logger, writers)
}

func TestLog_StandaloneWriteSurfacesBusyTimeout(t *testing.T) {
	db := setupFileDB(t, "20ms")
	logger, err := audit.New(db)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	blocker, err := db.Begin()
	if err != nil {
		t.Fatalf("begin blocker transaction: %v", err)
	}
	defer blocker.Rollback()

	err = logger.Log(nil, "notification_sent", "notification", "blocked", nil, map[string]bool{"blocked": true})
	if err == nil {
		t.Fatal("Log returned nil while another writer held the lock past the configured timeout")
	}

	if err := blocker.Rollback(); err != nil {
		t.Fatalf("release blocker transaction: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events").Scan(&count); err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("found %d audit rows after timed-out write, want 0", count)
	}

	if err := logger.Log(nil, "notification_sent", "notification", "recovered", nil, map[string]bool{"blocked": false}); err != nil {
		t.Fatalf("Log did not recover after lock release: %v", err)
	}
}
