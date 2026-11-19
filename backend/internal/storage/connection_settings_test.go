package storage

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// captureLog redirects the standard logger for the rest of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// twoConns reserves two distinct pooled connections at once.
func twoConns(t *testing.T, db *sql.DB) (*sql.Conn, *sql.Conn) {
	t.Helper()
	a, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("reserve first connection: %v", err)
	}
	t.Cleanup(func() { a.Close() })
	b, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("reserve second connection: %v", err)
	}
	t.Cleanup(func() { b.Close() })
	return a, b
}

func openBoth(t *testing.T) map[string]*Storage {
	t.Helper()
	file, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { file.DB.Close() })
	mem, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory: %v", err)
	}
	t.Cleanup(func() { mem.DB.Close() })
	return map[string]*Storage{"file": file, "in-memory": mem}
}

func TestOpen_EveryConnectionEnforcesForeignKeysAndBusyTimeout(t *testing.T) {
	t.Setenv("DB_BUSY_TIMEOUT", "1234ms")
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")

	for name, store := range openBoth(t) {
		t.Run(name, func(t *testing.T) {
			a, b := twoConns(t, store.DB)
			for i, conn := range []*sql.Conn{a, b} {
				var fk, timeout int
				if err := conn.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&fk); err != nil {
					t.Fatalf("conn %d: read foreign_keys: %v", i, err)
				}
				if fk != 1 {
					t.Errorf("conn %d: foreign_keys = %d, want 1", i, fk)
				}
				if err := conn.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&timeout); err != nil {
					t.Fatalf("conn %d: read busy_timeout: %v", i, err)
				}
				if timeout != 1234 {
					t.Errorf("conn %d: busy_timeout = %d, want 1234", i, timeout)
				}
				_, err := conn.ExecContext(context.Background(),
					`INSERT INTO sops_tags (sop_id, tag_id) VALUES ('missing-sop', 'missing-tag')`)
				if err == nil {
					t.Errorf("conn %d: insert with missing parent rows was accepted", i)
				}
			}
		})
	}
}

func TestOpen_TransactionHoldsWriteLockFromBegin(t *testing.T) {
	t.Setenv("DB_BUSY_TIMEOUT", "20ms")
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")

	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.DB.Close()

	first, err := store.DB.Begin()
	if err != nil {
		t.Fatalf("begin first transaction: %v", err)
	}
	defer first.Rollback()

	// The first transaction has not written anything yet.
	started := time.Now()
	second, err := store.DB.Begin()
	if err == nil {
		second.Rollback()
		t.Fatal("second transaction began while the first was open; the first did not hold the write lock")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "busy") && !strings.Contains(strings.ToLower(err.Error()), "locked") {
		t.Errorf("second Begin error = %v, want a busy error", err)
	}
	if waited := time.Since(started); waited < 15*time.Millisecond {
		t.Errorf("second Begin failed after %v, want it to wait for the busy timeout", waited)
	}
}

func TestOpen_BusyTimeoutResolution(t *testing.T) {
	cases := []struct {
		name        string
		db, audit   string
		wantMs      int
		wantWarning bool
	}{
		{name: "default", wantMs: 5000},
		{name: "DB_BUSY_TIMEOUT", db: "750ms", wantMs: 750},
		{name: "AUDIT_BUSY_TIMEOUT fallback", audit: "2s", wantMs: 2000, wantWarning: true},
		{name: "DB_BUSY_TIMEOUT wins", db: "300ms", audit: "2s", wantMs: 300},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DB_BUSY_TIMEOUT", tc.db)
			t.Setenv("AUDIT_BUSY_TIMEOUT", tc.audit)
			logs := captureLog(t)

			store, err := Open(t.TempDir())
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer store.DB.Close()

			var timeout int
			if err := store.DB.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
				t.Fatalf("read busy_timeout: %v", err)
			}
			if timeout != tc.wantMs {
				t.Errorf("busy_timeout = %d, want %d", timeout, tc.wantMs)
			}
			warned := strings.Contains(logs.String(), "AUDIT_BUSY_TIMEOUT is deprecated")
			if warned != tc.wantWarning {
				t.Errorf("deprecation warning logged = %v, want %v; log:\n%s", warned, tc.wantWarning, logs.String())
			}
		})
	}
}

func TestOpen_RejectsInvalidBusyTimeout(t *testing.T) {
	for _, tc := range []struct{ name, db, audit string }{
		{name: "not a duration", db: "not-a-duration"},
		{name: "below 1ms", db: "500us"},
		{name: "invalid fallback", audit: "soon"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DB_BUSY_TIMEOUT", tc.db)
			t.Setenv("AUDIT_BUSY_TIMEOUT", tc.audit)
			if store, err := Open(t.TempDir()); err == nil {
				store.DB.Close()
				t.Fatal("Open accepted an invalid busy timeout")
			}
			if store, err := OpenInMemory(); err == nil {
				store.DB.Close()
				t.Fatal("OpenInMemory accepted an invalid busy timeout")
			}
		})
	}
}

func TestOpen_WarnsAboutExistingForeignKeyViolations(t *testing.T) {
	t.Setenv("DB_BUSY_TIMEOUT", "")
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")
	dir := t.TempDir()

	store, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	store.DB.Close()

	// Write an orphan row the way old connections could: with enforcement off.
	raw, err := sql.Open("sqlite", filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(`PRAGMA foreign_keys = OFF; INSERT INTO sops_tags (sop_id, tag_id) VALUES ('missing-sop', 'missing-tag')`); err != nil {
		t.Fatalf("insert orphan row: %v", err)
	}
	raw.Close()

	logs := captureLog(t)
	store, err = Open(dir)
	if err != nil {
		t.Fatalf("Open with orphan rows must still succeed: %v", err)
	}
	defer store.DB.Close()

	var warnings []string
	for _, line := range strings.Split(logs.String(), "\n") {
		if strings.Contains(line, "WARNING") && strings.Contains(line, "foreign key") {
			warnings = append(warnings, line)
		}
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d foreign-key warnings, want 1 for sops_tags; log:\n%s", len(warnings), logs.String())
	}
	for _, want := range []string{"sops_tags", "1 row", "sops", "tags"} {
		if !strings.Contains(warnings[0], want) {
			t.Errorf("warning %q does not mention %q", warnings[0], want)
		}
	}
}

func TestOpen_NoForeignKeyWarningForCleanData(t *testing.T) {
	t.Setenv("DB_BUSY_TIMEOUT", "")
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")
	logs := captureLog(t)

	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.DB.Close()

	if strings.Contains(logs.String(), "WARNING") {
		t.Errorf("clean database logged a warning:\n%s", logs.String())
	}
}
