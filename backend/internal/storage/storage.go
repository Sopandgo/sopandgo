package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Storage struct {
	DB *sql.DB
}

const defaultBusyTimeout = 5 * time.Second

// Open initializes the SQLite database in the given data directory.
func Open(dataDir string) (*Storage, error) {
	return open(filepath.Join(dataDir, "app.db"), "database schema version: %d")
}

// OpenInMemory initializes a fresh SQLite database entirely in RAM.
// This is exclusively used for the testing environment.
func OpenInMemory() (*Storage, error) {
	// Use a shared memory URI so multiple connections in a pool can see the same tables
	return open("file::memory:?cache=shared", "in-memory database schema version initialized: %d")
}

// open applies the same connection settings to every pooled connection, in
// production and in tests:
//
//   - foreign keys on, so no connection accepts rows pointing at missing parents;
//   - a busy timeout, so a writer waits for another writer instead of failing at once;
//   - immediate transactions, so every transaction holds the write lock from begin.
//
// These are per-connection settings. The driver applies connection-string
// parameters to each connection it opens; a PRAGMA run on the pool would reach
// only one of them.
func open(dsn string, versionLogFormat string) (*Storage, error) {
	busyTimeout, err := busyTimeoutFromEnv()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn+connectionParams(dsn, busyTimeout))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// WAL is a persistent property of the database file, so setting it once is enough.
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		db.Close()
		return nil, err
	}

	version, err := runMigrations(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	log.Printf(versionLogFormat, version)

	if err := warnForeignKeyViolations(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Storage{DB: db}, nil
}

func connectionParams(dsn string, busyTimeout time.Duration) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_txlock=immediate",
		sep, busyTimeout.Milliseconds())
}

// busyTimeoutFromEnv reads DB_BUSY_TIMEOUT, falling back to the deprecated
// AUDIT_BUSY_TIMEOUT. DB_BUSY_TIMEOUT wins when both are set.
func busyTimeoutFromEnv() (time.Duration, error) {
	name := "DB_BUSY_TIMEOUT"
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		if legacy := strings.TrimSpace(os.Getenv("AUDIT_BUSY_TIMEOUT")); legacy != "" {
			name, raw = "AUDIT_BUSY_TIMEOUT", legacy
			log.Printf("WARNING: AUDIT_BUSY_TIMEOUT is deprecated and now applies to every database write; rename it to DB_BUSY_TIMEOUT in .env or docker-compose.yml")
		}
	}
	if raw == "" {
		return defaultBusyTimeout, nil
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < time.Millisecond {
		return 0, fmt.Errorf("%s must be a duration of at least 1ms", name)
	}
	return parsed, nil
}

// warnForeignKeyViolations reports rows written before enforcement was on
// every connection. SQLite checks only new writes, so this is the only place
// such rows surface. It logs and carries on so a lab keeps running.
func warnForeignKeyViolations(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("foreign key check: %w", err)
	}
	defer rows.Close()

	type violations struct {
		rows    map[int64]struct{}
		parents map[string]struct{}
	}
	byTable := map[string]*violations{}
	for rows.Next() {
		var (
			table, parent string
			rowID         sql.NullInt64
			fkID          int
		)
		if err := rows.Scan(&table, &rowID, &parent, &fkID); err != nil {
			return fmt.Errorf("foreign key check: %w", err)
		}
		v := byTable[table]
		if v == nil {
			v = &violations{rows: map[int64]struct{}{}, parents: map[string]struct{}{}}
			byTable[table] = v
		}
		v.rows[rowID.Int64] = struct{}{}
		v.parents[parent] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("foreign key check: %w", err)
	}

	tables := make([]string, 0, len(byTable))
	for table := range byTable {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		v := byTable[table]
		parents := make([]string, 0, len(v.parents))
		for parent := range v.parents {
			parents = append(parents, parent)
		}
		sort.Strings(parents)
		log.Printf("WARNING: foreign key violations in table %s: %d row(s) reference missing rows in %s",
			table, len(v.rows), strings.Join(parents, ", "))
	}
	return nil
}
