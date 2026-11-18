package audit

import (
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

// These vectors are also produced by docs/ops/verify_anchors.py. If one
// changes, every exported evidence bundle stops verifying.
func TestComputeHash_Vectors(t *testing.T) {
	actor := "user-123"
	base := HashInput{
		PrevHash:   GenesisHash,
		EventType:  "user_created",
		EntityType: "user",
		EntityID:   "1",
		Payload:    `{"name":"Alice"}`,
		CreatedAt:  "2026-11-18T10:00:00.123456789Z",
	}
	withActor := base
	withActor.ActorUserID = &actor

	cases := []struct {
		name    string
		version int
		in      HashInput
		want    string
	}{
		{"v1", HashV1, withActor, "5138f3d19128405dad31fba13596f2479534d4cf0b185076fc06442d5ec985f8"},
		{"v2 with actor", HashV2, withActor, "ec4f0361affac4711a49da38208bf68bdd854b484fd9f36c9ee5e925b075cf2c"},
		{"v2 null actor", HashV2, base, "bc62abd7a0130c6135a863783c0d29c55a8030a7efd782ce22fa60aa407376dc"},
	}
	for _, tc := range cases {
		got, err := ComputeHash(tc.version, tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}

	if _, err := ComputeHash(99, base); err == nil {
		t.Error("unknown version should fail")
	}
}

// v2 must not be fooled by moving bytes between fields, which v1 allows.
func TestComputeHash_V2FieldBoundaries(t *testing.T) {
	a := HashInput{PrevHash: GenesisHash, EventType: "ab", EntityID: "c", CreatedAt: "t"}
	b := HashInput{PrevHash: GenesisHash, EventType: "a", EntityID: "bc", CreatedAt: "t"}
	if hashV1(a) != hashV1(b) {
		t.Fatal("expected v1 collision for shifted field boundary")
	}
	if hashV2(a) == hashV2(b) {
		t.Fatal("v2 must separate field boundaries")
	}
}

func newHashTestLogger(t *testing.T) (*Logger, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT, entity_type TEXT, entity_id TEXT, actor_user_id TEXT,
			payload TEXT, created_at TEXT, hash TEXT, prev_hash TEXT,
			hash_version INTEGER NOT NULL DEFAULT 1
		);
		CREATE TABLE users (id TEXT PRIMARY KEY, display_name TEXT);
	`); err != nil {
		t.Fatal(err)
	}
	l, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return l, db
}

// A chain that starts with v1 rows (written before migration v30) and
// continues with v2 rows verifies as one chain.
func TestVerifyEntireChain_MixedVersions(t *testing.T) {
	l, db := newHashTestLogger(t)

	in := HashInput{PrevHash: GenesisHash, EventType: "system_restart", EntityType: "system", EntityID: "0", Payload: "{}", CreatedAt: "2025-01-01T00:00:00Z"}
	v1 := hashV1(in)
	if _, err := db.Exec(`INSERT INTO audit_events (event_type, entity_type, entity_id, payload, created_at, hash, prev_hash, hash_version)
		VALUES ('system_restart','system','0','{}','2025-01-01T00:00:00Z',?,?,1)`, v1, GenesisHash); err != nil {
		t.Fatal(err)
	}
	actor := "u1"
	if err := l.Log(nil, "login", "user", "u1", &actor, map[string]string{"ip": "x"}); err != nil {
		t.Fatal(err)
	}

	report, err := l.VerifyEntireChain()
	if err != nil {
		t.Fatal(err)
	}
	if report.Checked != 2 || report.Invalid != 0 {
		t.Fatalf("report = %+v", report)
	}
	var version int
	if err := db.QueryRow(`SELECT hash_version FROM audit_events WHERE id = 2`).Scan(&version); err != nil || version != HashV2 {
		t.Fatalf("new row version = %d, %v", version, err)
	}
}

// Changing who performed an action is detected on v2 rows.
func TestVerifyEntireChain_ActorTamperDetected(t *testing.T) {
	l, db := newHashTestLogger(t)
	actor := "alice"
	if err := l.Log(nil, "acknowledgment_added", "acknowledgment", "a1", &actor, "{}"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE audit_events SET actor_user_id = 'mallory' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	report, err := l.VerifyEntireChain()
	if err != nil {
		t.Fatal(err)
	}
	if report.Invalid != 1 || report.Failed[0].Error != "hash_mismatch" {
		t.Fatalf("report = %+v", report)
	}
}

func TestVerifySinceAndTip(t *testing.T) {
	l, db := newHashTestLogger(t)

	id, hash, err := l.Tip()
	if err != nil || id != 0 || hash != GenesisHash {
		t.Fatalf("empty tip = %d %s %v", id, hash, err)
	}

	for i := 0; i < 3; i++ {
		if err := l.Log(nil, "login", "user", "u", nil, "{}"); err != nil {
			t.Fatal(err)
		}
	}
	anchorID, anchorHash, err := l.Tip()
	if err != nil || anchorID != 3 {
		t.Fatalf("tip = %d %v", anchorID, err)
	}

	// Nothing new since the anchor.
	r, err := l.VerifySince(anchorID, anchorHash)
	if err != nil || r.Checked != 0 || r.TipID != anchorID || r.TipHash != anchorHash {
		t.Fatalf("idle VerifySince = %+v %v", r, err)
	}

	for i := 0; i < 2; i++ {
		if err := l.Log(nil, "logout", "user", "u", nil, "{}"); err != nil {
			t.Fatal(err)
		}
	}
	r, err = l.VerifySince(anchorID, anchorHash)
	if err != nil || r.Checked != 2 || r.Invalid != 0 || r.TipID != 5 {
		t.Fatalf("VerifySince = %+v %v", r, err)
	}

	// Rewriting the anchored row is reported as a moved anchor.
	if _, err := db.Exec(`UPDATE audit_events SET hash = 'forged' WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.VerifySince(anchorID, anchorHash); !errors.Is(err, ErrAnchorMoved) {
		t.Fatalf("expected ErrAnchorMoved, got %v", err)
	}

	// Deleting it is too.
	if _, err := db.Exec(`DELETE FROM audit_events WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.VerifySince(anchorID, anchorHash); !errors.Is(err, ErrAnchorMoved) {
		t.Fatalf("expected ErrAnchorMoved after delete, got %v", err)
	}
}
