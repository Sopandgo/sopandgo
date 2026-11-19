package sop_test

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// This test builds the SOP module on a file database opened by the storage
// package rather than the shared test environment. The in-memory test database
// uses shared-cache mode, which fails at once under contention instead of
// waiting, so it cannot stand in for concurrent writers.
func TestService_RegisterSOPVersion_ConcurrentCreatesGetDistinctVersions(t *testing.T) {
	t.Setenv("DB_BUSY_TIMEOUT", "5s")
	t.Setenv("AUDIT_BUSY_TIMEOUT", "")
	dataDir := t.TempDir()
	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })
	logger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("new audit logger: %v", err)
	}
	svc := sop.NewService(store.DB, logger, dataDir, "test-generator", false, nil)

	// Two editors, several saves each, all at once: enough overlap that
	// creates which did not serialize at begin would collide.
	editors := []string{"editor-a", "editor-b"}
	for _, id := range editors {
		if _, err := store.DB.Exec(`
			INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at)
			VALUES (?, ?, ?, 'hash', 'editor', 1, '2026-01-01T00:00:00Z')
		`, id, id, id+"@test.local"); err != nil {
			t.Fatalf("seed user %s: %v", id, err)
		}
	}
	sopID, err := svc.RegisterSOP("Concurrent Edits", &editors[0])
	if err != nil {
		t.Fatalf("register sop: %v", err)
	}

	const creates = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	versions := make([]int, creates)
	errs := make([]error, creates)
	for i := 0; i < creates; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			editor := editors[i%len(editors)]
			content := fmt.Sprintf("# Draft %d by %s", i, editor)
			_, versions[i], errs[i] = svc.RegisterSOPVersion(sopID, content, "Concurrent edit", &editor)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("create %d: RegisterSOPVersion: %v", i, err)
		}
	}
	sort.Ints(versions)
	for i, v := range versions {
		if v != i+1 {
			t.Fatalf("version numbers = %v, want 1..%d", versions, creates)
		}
	}

	stored, err := svc.GetSOPVersionsBySOPID(sopID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(stored) != creates {
		t.Fatalf("stored %d versions, want %d", len(stored), creates)
	}
	report, err := logger.VerifyEntireChain()
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if report.Invalid != 0 {
		t.Errorf("audit chain has %d invalid events: %+v", report.Invalid, report.Failed)
	}
}
