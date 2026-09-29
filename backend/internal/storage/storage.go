package storage

import (
	"database/sql"
	"log"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Storage struct {
	DB *sql.DB
}

// Open initializes the SQLite database in the given data directory.
func Open(dataDir string) (*Storage, error) {
	dbPath := filepath.Join(dataDir, "app.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if err := configureSQLite(db); err != nil {
		db.Close()
		return nil, err
	}

	version, err := runMigrations(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("database schema version: %d", version)

	return &Storage{DB: db}, nil
}

// OpenInMemory initializes a fresh SQLite database entirely in RAM.
// This is exclusively used for the testing environment.
func OpenInMemory() (*Storage, error) {
	// Use a shared memory URI so multiple connections in a pool can see the same tables
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if err := configureSQLite(db); err != nil {
		db.Close()
		return nil, err
	}

	version, err := runMigrations(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("in-memory database schema version initialized: %d", version)

	return &Storage{DB: db}, nil
}

func configureSQLite(db *sql.DB) error {
	_, err := db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA foreign_keys = ON;
	`)
	return err
}
