package storage

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsUniqueViolation(t *testing.T) {
	store, err := OpenInMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.DB.Close()

	if _, err := store.DB.Exec(`CREATE TABLE constraint_probe (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL UNIQUE,
		parent TEXT REFERENCES constraint_probe(id)
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := store.DB.Exec(`INSERT INTO constraint_probe (id, title) VALUES ('a', 'one')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	insert := func(query string) error {
		_, err := store.DB.Exec(query)
		if err == nil {
			t.Fatalf("%s: want an error", query)
		}
		return err
	}

	unique := insert(`INSERT INTO constraint_probe (id, title) VALUES ('b', 'one')`)
	if !IsUniqueViolation(unique) {
		t.Errorf("duplicate UNIQUE column: want true, got false for %v", unique)
	}
	if !IsUniqueViolation(fmt.Errorf("record acknowledgment: %w", unique)) {
		t.Errorf("a wrapped unique violation must still be detected")
	}

	primary := insert(`INSERT INTO constraint_probe (id, title) VALUES ('a', 'two')`)
	if !IsUniqueViolation(primary) {
		t.Errorf("duplicate primary key: want true, got false for %v", primary)
	}

	notNull := insert(`INSERT INTO constraint_probe (id, title) VALUES ('c', NULL)`)
	if IsUniqueViolation(notNull) {
		t.Errorf("NOT NULL violation must not count as a duplicate: %v", notNull)
	}

	foreignKey := insert(`INSERT INTO constraint_probe (id, title, parent) VALUES ('d', 'four', 'missing')`)
	if IsUniqueViolation(foreignKey) {
		t.Errorf("foreign key violation must not count as a duplicate: %v", foreignKey)
	}

	if IsUniqueViolation(errors.New("UNIQUE constraint failed: but only text")) {
		t.Errorf("an error that only says UNIQUE must not count")
	}
	if IsUniqueViolation(nil) {
		t.Errorf("nil is not a violation")
	}
}
