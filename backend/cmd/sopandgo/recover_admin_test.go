package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// seedAdmin creates an on-disk database with one admin, like a live DATA_DIR.
func seedAdmin(t *testing.T, dataDir string) {
	t.Helper()
	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	logger, err := audit.New(store.DB)
	if err != nil {
		t.Fatal(err)
	}
	svc := auth.NewService(store.DB, logger, dataDir, nil)
	id, err := svc.RegisterUser("Lab Admin", "admin@lab.test", auth.RoleAdmin, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateUserPassword(id, "OriginalPassword1!", nil); err != nil {
		t.Fatal(err)
	}
}

func countBreakGlassEvents(t *testing.T, dataDir, eventType string) int {
	t.Helper()
	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.DB.Close()
	var n int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = ?`, eventType).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRecoverAdmin_ResetsAndAudits(t *testing.T) {
	dataDir := t.TempDir()
	seedAdmin(t, dataDir)
	t.Setenv("DATA_DIR", dataDir)

	var out, errOut bytes.Buffer
	code := recoverAdmin([]string{"--email", "Admin@Lab.test"}, strings.NewReader("admin@lab.test\n"), &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Temporary password (shown once): ") {
		t.Fatalf("output: %s", out.String())
	}
	if n := countBreakGlassEvents(t, dataDir, audit.EventAdminBreakGlassReset); n != 1 {
		t.Fatalf("expected one admin_break_glass_reset event, got %d", n)
	}
}

func TestRecoverAdmin_MismatchedConfirmationChangesNothing(t *testing.T) {
	dataDir := t.TempDir()
	seedAdmin(t, dataDir)
	t.Setenv("DATA_DIR", dataDir)

	var out, errOut bytes.Buffer
	code := recoverAdmin([]string{"--email", "admin@lab.test"}, strings.NewReader("someone-else@lab.test\n"), &out, &errOut)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if countBreakGlassEvents(t, dataDir, audit.EventAdminBreakGlassReset) != 0 {
		t.Fatal("an aborted run must not reset anything")
	}
	if countBreakGlassEvents(t, dataDir, audit.EventAdminBreakGlassFailed) != 1 {
		t.Fatal("an aborted run must be audited")
	}
}

func TestRecoverAdmin_RefusesMissingDatabase(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	var out, errOut bytes.Buffer
	if code := recoverAdmin([]string{"--email", "admin@lab.test", "--yes"}, strings.NewReader(""), &out, &errOut); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "app.db")); !os.IsNotExist(err) {
		t.Fatal("recover-admin must not create a database")
	}
}

func TestRecoverAdmin_ListAndUsage(t *testing.T) {
	dataDir := t.TempDir()
	seedAdmin(t, dataDir)
	t.Setenv("DATA_DIR", dataDir)

	var out, errOut bytes.Buffer
	if code := recoverAdmin([]string{"--list"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("list exit %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "admin@lab.test") {
		t.Fatalf("list output: %s", out.String())
	}
	if code := recoverAdmin(nil, strings.NewReader(""), &out, &errOut); code != 2 {
		t.Fatalf("no flags should be a usage error, got %d", code)
	}
}
