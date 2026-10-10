package main

import (
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
		{value: "", want: true},
		{value: "true", want: true},
		{value: "TRUE", want: true},
		{value: "false", want: false},
		{value: "FALSE", want: false},
		{value: " false ", want: false},
		{value: "0", want: true},
		{value: "no", want: true},
	}

	for _, tc := range cases {
		if got := seedDemoEnabled(tc.value); got != tc.want {
			t.Errorf("seedDemoEnabled(%q) = %v, want %v", tc.value, got, tc.want)
		}
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

	if err := seed(dataDir, sopsRoot); err != nil {
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
	authService := auth.NewService(store.DB, auditLogger, dataDir)

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

	sops, total, err := sopService.ListSOPs(researcher.ID, 100, 0, "", "", false, false)
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
	if len(tags) != 4 {
		t.Errorf("seeded %d tags, want 4", len(tags))
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
	if err := seed(dataDir, sopsRoot); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	_, total, err = sopService.ListSOPs(researcher.ID, 100, 0, "", "", false, false)
	if err != nil {
		t.Fatalf("ListSOPs after second seed: %v", err)
	}
	if total != 6 {
		t.Errorf("after second seed: %d SOPs, want 6", total)
	}
}
