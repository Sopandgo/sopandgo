package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func TestSeedDemoEnabled(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{value: "", want: false},
		{value: "true", want: true},
		{value: "TRUE", want: true},
		{value: " True ", want: true},
		{value: "false", want: false},
		{value: "FALSE", want: false},
		{value: "0", want: false},
		{value: "1", want: false},
		{value: "yes", want: false},
	}

	for _, tc := range cases {
		if got := seedDemoEnabled(tc.value); got != tc.want {
			t.Errorf("seedDemoEnabled(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// TestRunSeedsOnlyWhenAsked: the seeder binary decides on its own whether to
// seed. Only SEED_DEMO_DATA=true creates the database.
func TestRunSeedsOnlyWhenAsked(t *testing.T) {
	sopsRoot := filepath.Join("..", "..", "demo", "sops")
	for _, value := range []string{"", "false", "no"} {
		dataDir := t.TempDir()
		if err := run(settings{SeedDemoData: value}, dataDir, sopsRoot); err != nil {
			t.Fatalf("run(SEED_DEMO_DATA=%q): %v", value, err)
		}
		if _, err := os.Stat(filepath.Join(dataDir, "app.db")); !os.IsNotExist(err) {
			t.Errorf("SEED_DEMO_DATA=%q created app.db (stat err %v)", value, err)
		}
	}

	dataDir := t.TempDir()
	if err := run(settings{SeedDemoData: " TRUE "}, dataDir, sopsRoot); err != nil {
		t.Fatalf("run(SEED_DEMO_DATA=TRUE): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "app.db")); err != nil {
		t.Errorf("SEED_DEMO_DATA=TRUE did not create app.db: %v", err)
	}
}

func TestParseManifestDefaults(t *testing.T) {
	m, err := parseManifest(nil, 2)
	if err != nil {
		t.Fatalf("parseManifest(nil) error: %v", err)
	}
	if m.FinalState != sop.StatePublished {
		t.Errorf("FinalState = %q, want %q", m.FinalState, sop.StatePublished)
	}
}

func TestWithDemoTag(t *testing.T) {
	got := withDemoTag([]string{"Safety", "demo", "Equipment"})
	want := []string{"Demo", "Safety", "Equipment"}
	if len(got) != len(want) {
		t.Fatalf("withDemoTag = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("withDemoTag = %v, want %v", got, want)
		}
	}
}

func TestParseManifestRejectsInvalid(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		versions int
		wantErr  string
	}{
		{"no versions", `{}`, 0, "no version-N.md"},
		{"unknown field", `{"tag": ["Safety"]}`, 1, "unknown field"},
		{"bad final state", `{"final_state": "approved"}`, 1, "final_state"},
		{"too many summaries", `{"change_summaries": ["a", "b"]}`, 1, "change_summaries"},
		{"unknown reader", `{"read_by": {"admin": [1]}}`, 1, "unknown demo user"},
		{"read unpublished rc", `{"final_state": "rc", "read_by": {"researcher": [2]}}`, 2, "never published"},
		{"read draft-only sop", `{"final_state": "draft", "read_by": {"researcher": [1]}}`, 1, "never published"},
		{"read version zero", `{"read_by": {"researcher": [0]}}`, 1, "never published"},
		{"unknown favorite", `{"favorited_by": ["admin"]}`, 1, "unknown demo user"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseManifest([]byte(tc.body), tc.versions)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseManifest error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestSeedShippedDemoData seeds the real backend/demo folder, so a broken
// sop.json or Markdown file fails here instead of at a lab's first boot.
func TestSeedShippedDemoData(t *testing.T) {
	dataDir := t.TempDir()
	sopsRoot := filepath.Join("..", "..", "demo", "sops")

	if err := seed(dataDir, sopsRoot, false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })

	auditLogger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("audit.New: %v", err)
	}
	sopService := sop.NewService(store.DB, auditLogger, dataDir, "1", false, nil)
	authService := auth.NewService(store.DB, auditLogger, dataDir, nil)

	researcher, _, err := authService.GetUserByEmail("researcher@demo.local")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if !researcher.HasAvatar {
		t.Fatal("expected researcher demo avatar")
	}
	for _, email := range []string{"manager@demo.local", "qa@demo.local"} {
		u, _, err := authService.GetUserByEmail(email)
		if err != nil {
			t.Fatalf("GetUserByEmail %s: %v", email, err)
		}
		if !u.HasAvatar {
			t.Fatalf("expected avatar for %s", email)
		}
	}

	sops, total, err := sopService.ListSOPs(researcher.ID, 100, 0, "", "", false, false, sop.ArchiveFilterActive)
	if err != nil {
		t.Fatalf("ListSOPs: %v", err)
	}
	if total != 6 {
		t.Errorf("seeded %d SOPs, want 6", total)
	}

	tags, err := sopService.ListTags()
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 5 {
		t.Errorf("seeded %d tags, want 5", len(tags))
	}
	var demoTagID string
	for _, tag := range tags {
		if tag.Title == "Demo" {
			demoTagID = tag.ID
			break
		}
	}
	if demoTagID == "" {
		t.Fatal("expected Demo tag among seeded tags")
	}
	for _, item := range sops {
		hasDemo := false
		for _, tag := range item.Tags {
			if tag.ID == demoTagID {
				hasDemo = true
				break
			}
		}
		if !hasDemo {
			t.Errorf("SOP %q missing Demo tag", item.Title)
		}
	}

	// The researcher's dashboard shows one never-signed SOP and one that
	// needs re-signing after an update.
	statuses, err := sopService.GetSignatureStatusByUser(researcher.ID)
	if err != nil {
		t.Fatalf("GetSignatureStatusByUser: %v", err)
	}
	pending := map[string]bool{}
	for _, st := range statuses {
		if !st.HasSignedLatest {
			pending[st.Title] = st.SignedOlderVersion
		}
	}
	wantPending := map[string]bool{
		"Pipette Performance Check (Demo)": false,
		"Chemical Spill Response (Demo)":   true,
	}
	if len(pending) != len(wantPending) {
		t.Errorf("researcher pending = %v, want %v", pending, wantPending)
	}
	for title, signedOlder := range wantPending {
		got, ok := pending[title]
		if !ok || got != signedOlder {
			t.Errorf("pending[%q] = %v (present %v), want signed older %v", title, got, ok, signedOlder)
		}
	}

	// The last version of each SOP stops at the state its sop.json asks for.
	wantLatest := map[string]string{
		"Autoclave Operation (Demo)":                  sop.StateRC,
		"Bacterial Glycerol Stock Preparation (Demo)": sop.StateDraft,
		"Lab Waste Segregation (Demo)":                sop.StatePublished,
	}
	for _, item := range sops {
		want, ok := wantLatest[item.Title]
		if !ok {
			continue
		}
		versions, err := sopService.GetSOPVersionsBySOPID(item.ID)
		if err != nil {
			t.Fatalf("GetSOPVersionsBySOPID: %v", err)
		}
		latest := versions[0]
		for _, v := range versions {
			if v.Version > latest.Version {
				latest = v
			}
		}
		if latest.Status != want {
			t.Errorf("%s latest status = %q, want %q", item.Title, latest.Status, want)
		}
		delete(wantLatest, item.Title)
	}
	if len(wantLatest) > 0 {
		t.Errorf("SOPs not found in library: %v", wantLatest)
	}

	// A second run must not touch an existing database.
	if err := seed(dataDir, sopsRoot, false); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	_, total, err = sopService.ListSOPs(researcher.ID, 100, 0, "", "", false, false, sop.ArchiveFilterActive)
	if err != nil {
		t.Fatalf("ListSOPs after second seed: %v", err)
	}
	if total != 6 {
		t.Errorf("after second seed: %d SOPs, want 6", total)
	}
}

const shippedSOPs = "../../demo/sops"

func openSeeded(t *testing.T, dataDir string) (*auth.Service, *sop.Service) {
	t.Helper()
	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })
	auditLogger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("audit.New: %v", err)
	}
	return auth.NewService(store.DB, auditLogger, dataDir, nil),
		sop.NewService(store.DB, auditLogger, dataDir, "1", false, nil)
}

func countSOPs(t *testing.T, dataDir string) int {
	t.Helper()
	_, sopService := openSeeded(t, dataDir)
	_, total, err := sopService.ListSOPs("", 100, 0, "", "", false, false, sop.ArchiveFilterActive)
	if err != nil {
		t.Fatalf("ListSOPs: %v", err)
	}
	return total
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestRunDemoModeSeedsEmptyDirectory(t *testing.T) {
	dataDir := t.TempDir()
	if err := run(settings{DemoMode: "true"}, dataDir, shippedSOPs); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !exists(filepath.Join(dataDir, demoMarker)) {
		t.Fatal("demo mode did not write the marker")
	}
	if got := countSOPs(t, dataDir); got != 6 {
		t.Fatalf("seeded %d SOPs, want 6", got)
	}
}

func TestRunDemoModeWipesMarkedDirectory(t *testing.T) {
	dataDir := t.TempDir()
	if err := run(settings{DemoMode: "TRUE"}, dataDir, shippedSOPs); err != nil {
		t.Fatalf("first run: %v", err)
	}

	// A visitor day leaves rows and files behind. The database is closed
	// again before the reset, as it is when the container restarts.
	store, err := storage.Open(dataDir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	auditLogger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("audit.New: %v", err)
	}
	if _, err := sop.NewService(store.DB, auditLogger, dataDir, "1", false, nil).RegisterSOP("Left behind", nil); err != nil {
		t.Fatalf("RegisterSOP: %v", err)
	}
	store.DB.Close()
	if err := os.MkdirAll(filepath.Join(dataDir, "uploads"), 0o755); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(dataDir, "uploads", "stray.png")
	if err := os.WriteFile(stray, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run(settings{DemoMode: "true"}, dataDir, shippedSOPs); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if exists(stray) {
		t.Error("reset kept a file from the previous cycle")
	}
	if !exists(filepath.Join(dataDir, demoMarker)) {
		t.Error("reset did not write the marker again")
	}
	if got := countSOPs(t, dataDir); got != 6 {
		t.Errorf("after reset: %d SOPs, want the 6 shipped ones", got)
	}
}

func TestRunDemoModeRefusesRealData(t *testing.T) {
	dataDir := t.TempDir()
	if err := run(settings{SeedDemoData: "true"}, dataDir, shippedSOPs); err != nil {
		t.Fatalf("normal seed: %v", err)
	}
	labFile := filepath.Join(dataDir, "lab-notes.txt")
	if err := os.WriteFile(labFile, []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run(settings{DemoMode: "true"}, dataDir, shippedSOPs)
	if err == nil || !strings.Contains(err.Error(), demoMarker) {
		t.Fatalf("run on real data = %v, want a refusal naming %s", err, demoMarker)
	}
	if !exists(labFile) || !exists(filepath.Join(dataDir, "app.db")) {
		t.Fatal("refused run still changed the data directory")
	}
	if exists(filepath.Join(dataDir, demoMarker)) {
		t.Fatal("refused run wrote the marker")
	}
}

func TestRunDemoModeRefusesExplicitNoSeed(t *testing.T) {
	for _, value := range []string{"false", " FALSE "} {
		dataDir := t.TempDir()
		err := run(settings{DemoMode: "true", SeedDemoData: value}, dataDir, shippedSOPs)
		if err == nil || !strings.Contains(err.Error(), "SEED_DEMO_DATA") {
			t.Errorf("DEMO_MODE with SEED_DEMO_DATA=%q = %v, want a refusal", value, err)
		}
		if exists(filepath.Join(dataDir, "app.db")) {
			t.Errorf("refused run created app.db")
		}
	}
}

func TestDemoAuditorOnlyInDemoMode(t *testing.T) {
	demoDir, normalDir := t.TempDir(), t.TempDir()
	if err := run(settings{DemoMode: "true"}, demoDir, shippedSOPs); err != nil {
		t.Fatalf("demo run: %v", err)
	}
	if err := run(settings{SeedDemoData: "true"}, normalDir, shippedSOPs); err != nil {
		t.Fatalf("normal run: %v", err)
	}

	demoAuth, _ := openSeeded(t, demoDir)
	auditor, _, err := demoAuth.GetUserByEmail("auditor@demo.local")
	if err != nil || auditor.Role != auth.RoleAuditor || auditor.DisplayName != "Demo Auditor" {
		t.Fatalf("demo auditor = %+v, %v; want Demo Auditor with role auditor", auditor, err)
	}
	normalAuth, _ := openSeeded(t, normalDir)
	if _, _, err := normalAuth.GetUserByEmail("auditor@demo.local"); err == nil {
		t.Fatal("normal seed created the demo auditor")
	}
}
