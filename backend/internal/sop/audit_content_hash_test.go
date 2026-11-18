package sop_test

import (
	"encoding/json"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

// External anchoring covers SOP content only through the audit chain, so every
// event that creates or signs a version must carry that version's content hash.
func TestAuditPayloads_CarryVersionContentHash(t *testing.T) {
	env := testenv.New(t)
	approverID := "hash-approver"
	readerID := "hash-reader"
	env.SeedTestUser(t, approverID, "approver@hash.local", auth.RoleApprover)
	env.SeedTestUser(t, readerID, "reader@hash.local", auth.RoleViewer)

	_, versionID := publishedSOP(t, env, "Pipette calibration", approverID)
	if _, err := env.SOPService.AddAcknowledgment(versionID, readerID, sop.AckTypeRead); err != nil {
		t.Fatalf("AddAcknowledgment: %v", err)
	}

	var want string
	if err := env.Store.DB.QueryRow(`SELECT content_hash FROM sop_versions WHERE id = ?`, versionID).Scan(&want); err != nil || want == "" {
		t.Fatalf("content_hash = %q, %v", want, err)
	}

	rows, err := env.Store.DB.Query(`
		SELECT event_type, payload FROM audit_events
		WHERE event_type IN ('sop_version_created', 'sop_acknowledgment_created', 'acknowledgment_added')
		ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	seen := map[string]int{}
	for rows.Next() {
		var eventType, payload string
		if err := rows.Scan(&eventType, &payload); err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Fatalf("%s payload: %v", eventType, err)
		}
		if p["content_hash"] != want {
			t.Errorf("%s payload content_hash = %v, want %s", eventType, p["content_hash"], want)
		}
		seen[eventType]++
	}
	// Created, author ack, approver ack, reader ack.
	if seen["sop_version_created"] != 1 || seen["sop_acknowledgment_created"] != 2 || seen["acknowledgment_added"] != 1 {
		t.Fatalf("events seen = %v", seen)
	}
}
