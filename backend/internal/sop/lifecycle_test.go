package sop_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"github.com/sopandgo/sopandgo/backend/internal/sop"
	"github.com/sopandgo/sopandgo/backend/internal/testenv"
)

const (
	lifecycleActor  = "lifecycle-actor"
	lifecycleReader = "lifecycle-reader"
)

func newLifecycleEnv(t *testing.T) *testenv.Env {
	t.Helper()
	env := testenv.New(t)
	env.SeedTestUser(t, lifecycleActor, "lifecycle@test.local", auth.RoleAdmin)
	env.SeedTestUser(t, lifecycleReader, "lifecycle-reader@test.local", auth.RoleViewer)
	return env
}

// auditEntry is an audit event as an auditor reads it.
type auditEntry struct {
	eventType  string
	entityType string
	payload    map[string]any
}

func (e auditEntry) String() string {
	return fmt.Sprintf("%s/%s %v", e.eventType, e.entityType, e.payload)
}

// versionEvents returns the audit events that name a version, oldest first.
func versionEvents(t *testing.T, env *testenv.Env, versionID string) []auditEntry {
	t.Helper()
	events, _, err := env.AuditLogger.List(1000, 0, audit.ListFilters{SOPID: versionID})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	out := make([]auditEntry, 0, len(events))
	for _, ev := range events {
		var payload map[string]any
		if err := json.Unmarshal([]byte(ev.Payload), &payload); err != nil {
			t.Fatalf("audit payload %q: %v", ev.Payload, err)
		}
		out = append(out, auditEntry{eventType: ev.EventType, entityType: ev.EntityType, payload: payload})
	}
	return out
}

func versionAcks(t *testing.T, env *testenv.Env, versionID string) []sop.SOPAcknowledgment {
	t.Helper()
	acks, err := env.SOPService.GetAcknowledgmentsByVersion(versionID)
	if err != nil {
		t.Fatalf("list acknowledgments: %v", err)
	}
	return acks
}

func versionStatus(t *testing.T, env *testenv.Env, versionID string) string {
	t.Helper()
	v, err := env.SOPService.GetSOPVersionByID(versionID)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	return v.Status
}

func versionCount(t *testing.T, env *testenv.Env, sopID string) int {
	t.Helper()
	versions, err := env.SOPService.GetSOPVersionsBySOPID(sopID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	return len(versions)
}

// snapshot is what an outsider can see of a version and its SOP.
type snapshot struct {
	status   string
	acks     int
	events   int
	versions int
}

func takeSnapshot(t *testing.T, env *testenv.Env, sopID, versionID string) snapshot {
	t.Helper()
	return snapshot{
		status:   versionStatus(t, env, versionID),
		acks:     len(versionAcks(t, env, versionID)),
		events:   len(versionEvents(t, env, versionID)),
		versions: versionCount(t, env, sopID),
	}
}

func requireConflict(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.HasPrefix(err.Error(), "conflict:") {
		t.Fatalf("want a conflict error, got %v", err)
	}
}

func strPtr(s string) *string { return &s }

var contentCounter int

// newVersion registers a draft with content no other version has.
func newDraft(t *testing.T, env *testenv.Env, sopID string) string {
	t.Helper()
	contentCounter++
	id, _, err := env.SOPService.RegisterSOPVersion(sopID, fmt.Sprintf("# Step %d", contentCounter), "Change", lifecycleActor)
	if err != nil {
		t.Fatalf("RegisterSOPVersion: %v", err)
	}
	return id
}

func mustDo(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// versionIn builds an SOP with a version in the given starting state. "archived"
// is a published version of an archived SOP.
func versionIn(t *testing.T, env *testenv.Env, state string) (sopID, versionID string) {
	t.Helper()
	sopID, err := env.SOPService.RegisterSOP("Lifecycle "+state, strPtr(lifecycleActor))
	mustDo(t, "RegisterSOP", err)
	versionID = newDraft(t, env, sopID)
	if state == sop.StateDraft {
		return sopID, versionID
	}
	mustDo(t, "promote", env.SOPService.PromoteSOPVersion(versionID, lifecycleActor))
	switch state {
	case sop.StateRC:
		return sopID, versionID
	case sop.StateRejected:
		mustDo(t, "reject", env.SOPService.RejectSOPVersion(versionID, lifecycleActor, "Not yet"))
		return sopID, versionID
	}
	_, err = env.SOPService.ApproveSOPVersion(versionID, lifecycleActor)
	mustDo(t, "approve", err)
	switch state {
	case sop.StateSuperseded:
		next := newDraft(t, env, sopID)
		mustDo(t, "promote next", env.SOPService.PromoteSOPVersion(next, lifecycleActor))
		_, err = env.SOPService.ApproveSOPVersion(next, lifecycleActor)
		mustDo(t, "approve next", err)
	case "archived":
		mustDo(t, "archive", env.SOPService.ArchiveSOP(sopID, lifecycleActor, "Retired"))
	}
	return sopID, versionID
}

// lifecycleOp is one public SOP operation, run against a version or its SOP.
// It returns the version it changed.
type lifecycleOp func(env *testenv.Env, sopID, versionID string) (string, error)

var lifecycleOps = map[string]lifecycleOp{
	"create": func(env *testenv.Env, sopID, _ string) (string, error) {
		contentCounter++
		id, _, err := env.SOPService.RegisterSOPVersion(sopID, fmt.Sprintf("# New %d", contentCounter), "New", lifecycleActor)
		return id, err
	},
	"promote": func(env *testenv.Env, _, versionID string) (string, error) {
		return versionID, env.SOPService.PromoteSOPVersion(versionID, lifecycleActor)
	},
	"reject": func(env *testenv.Env, _, versionID string) (string, error) {
		return versionID, env.SOPService.RejectSOPVersion(versionID, lifecycleActor, "  Step 2 is unsafe  ")
	},
	"approve": func(env *testenv.Env, _, versionID string) (string, error) {
		_, err := env.SOPService.ApproveSOPVersion(versionID, lifecycleActor)
		return versionID, err
	},
	"reader acknowledgment": func(env *testenv.Env, _, versionID string) (string, error) {
		_, err := env.SOPService.AddAcknowledgment(versionID, lifecycleReader, sop.AckTypeRead)
		return versionID, err
	},
}

// allowed describes what an allowed operation leaves behind on the version it changed.
type allowed struct {
	status string
	// ack is the acknowledgment type added, or "" for none.
	ack string
	// events are the audit events added, as "event_type/entity_type new_state".
	events []string
}

var (
	createsDraft = &allowed{status: sop.StateDraft, ack: sop.AckTypeAuthor, events: []string{
		"sop_version_created/sop_version",
		"sop_acknowledgment_created/sop_version author",
	}}
	refused  = (*allowed)(nil)
	archived = &allowed{status: "archived"}
)

func TestLifecycle_EveryOperationFromEveryState(t *testing.T) {
	table := map[string]map[string]*allowed{
		sop.StateDraft: {
			"create":                createsDraft,
			"promote":               {status: sop.StateRC, events: []string{"sop_version_state_changed/sop_version rc"}},
			"reject":                refused,
			"approve":               refused,
			"reader acknowledgment": refused,
		},
		sop.StateRC: {
			"create":  refused,
			"promote": refused,
			"reject":  {status: sop.StateRejected, events: []string{"sop_version_state_changed/sop_version rejected"}},
			"approve": {status: sop.StatePublished, ack: sop.AckTypeApproved, events: []string{
				"sop_acknowledgment_created/sop_version approver",
				"sop_version_state_changed/sop_version published",
			}},
			"reader acknowledgment": refused,
		},
		sop.StatePublished: {
			"create":  createsDraft,
			"promote": refused,
			"reject":  refused,
			"approve": refused,
			"reader acknowledgment": {status: sop.StatePublished, ack: sop.AckTypeRead, events: []string{
				"acknowledgment_added/acknowledgment reader",
			}},
		},
		sop.StateRejected: {
			"create":                createsDraft,
			"promote":               refused,
			"reject":                refused,
			"approve":               refused,
			"reader acknowledgment": refused,
		},
		sop.StateSuperseded: {
			"create":                createsDraft,
			"promote":               refused,
			"reject":                refused,
			"approve":               refused,
			"reader acknowledgment": refused,
		},
		"archived": {
			"create":                archived,
			"promote":               archived,
			"reject":                archived,
			"approve":               archived,
			"reader acknowledgment": archived,
		},
	}

	for state, row := range table {
		for opName, want := range row {
			t.Run(state+"/"+opName, func(t *testing.T) {
				env := newLifecycleEnv(t)
				sopID, versionID := versionIn(t, env, state)
				before := takeSnapshot(t, env, sopID, versionID)
				beforeAcks := map[string]bool{}
				for _, a := range versionAcks(t, env, versionID) {
					beforeAcks[a.ID] = true
				}

				changedID, err := lifecycleOps[opName](env, sopID, versionID)

				switch want {
				case refused, archived:
					if want == archived {
						if !errors.Is(err, sop.ErrSOPArchived) {
							t.Fatalf("want ErrSOPArchived, got %v", err)
						}
					} else {
						requireConflict(t, err)
					}
					if after := takeSnapshot(t, env, sopID, versionID); after != before {
						t.Fatalf("a refused %s changed the record: before %+v, after %+v", opName, before, after)
					}
					return
				}

				if err != nil {
					t.Fatalf("%s from %s: %v", opName, state, err)
				}
				if got := versionStatus(t, env, changedID); got != want.status {
					t.Errorf("status: want %q, got %q", want.status, got)
				}

				var newAcks []string
				for _, a := range versionAcks(t, env, changedID) {
					if !beforeAcks[a.ID] {
						newAcks = append(newAcks, a.AcknowledgmentType)
					}
				}
				wantAcks := []string(nil)
				if want.ack != "" {
					wantAcks = []string{want.ack}
				}
				if fmt.Sprint(newAcks) != fmt.Sprint(wantAcks) {
					t.Errorf("acknowledgments added: want %v, got %v", wantAcks, newAcks)
				}

				events := versionEvents(t, env, changedID)
				if opName != "create" {
					events = events[before.events:]
				}
				var got []string
				for _, e := range events {
					label := e.eventType + "/" + e.entityType
					if s, ok := e.payload["new_state"]; ok {
						label += fmt.Sprintf(" %v", s)
					}
					if s, ok := e.payload["ack_type"]; ok {
						label += fmt.Sprintf(" %v", s)
					}
					got = append(got, label)
				}
				if strings.Join(got, "\n") != strings.Join(want.events, "\n") {
					t.Errorf("audit events added:\nwant %v\ngot  %v", want.events, got)
				}
			})
		}
	}
}

func TestLifecycle_RejectStoresTheTrimmedReasonInTheAuditEvent(t *testing.T) {
	env := newLifecycleEnv(t)
	_, vID := versionIn(t, env, sop.StateRC)

	mustDo(t, "reject", env.SOPService.RejectSOPVersion(vID, lifecycleActor, "  Step 2 is unsafe  "))

	events := versionEvents(t, env, vID)
	last := events[len(events)-1]
	if last.payload["new_state"] != sop.StateRejected || last.payload["reason"] != "Step 2 is unsafe" {
		t.Fatalf("want the rejection with its trimmed reason, got %s", last)
	}
}

func TestLifecycle_ApproveSupersedesThePublishedVersionInTodaysAuditOrder(t *testing.T) {
	env := newLifecycleEnv(t)
	sopID, v1 := versionIn(t, env, sop.StatePublished)
	v2 := newDraft(t, env, sopID)
	mustDo(t, "promote v2", env.SOPService.PromoteSOPVersion(v2, lifecycleActor))
	v1Before := len(versionEvents(t, env, v1))
	v2Before := len(versionEvents(t, env, v2))
	allBefore, _, err := env.AuditLogger.List(1, 0, audit.ListFilters{})
	mustDo(t, "list", err)
	lastIDBefore := allBefore[0].ID

	_, err = env.SOPService.ApproveSOPVersion(v2, lifecycleActor)
	mustDo(t, "approve v2", err)

	if got := versionStatus(t, env, v1); got != sop.StateSuperseded {
		t.Errorf("v1: want superseded, got %q", got)
	}
	if got := versionStatus(t, env, v2); got != sop.StatePublished {
		t.Errorf("v2: want published, got %q", got)
	}
	if got := versionEvents(t, env, v1)[v1Before:]; len(got) != 1 ||
		got[0].payload["new_state"] != sop.StateSuperseded || got[0].payload["reason"] != "Superseded by version 2" {
		t.Errorf("v1 events: want one supersede with its reason, got %v", got)
	}
	if got := versionEvents(t, env, v2)[v2Before:]; len(got) != 2 {
		t.Errorf("v2 events: want the approver acknowledgment and the published state, got %v", got)
	}

	// The audit log reads: supersede v1, approver acknowledgment, published v2.
	all, _, err := env.AuditLogger.List(10, 0, audit.ListFilters{})
	mustDo(t, "list", err)
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	var order []string
	for _, ev := range all {
		if ev.ID <= lastIDBefore {
			continue
		}
		order = append(order, ev.EntityID+" "+ev.EventType)
	}
	want := []string{
		v1 + " sop_version_state_changed",
		v2 + " sop_acknowledgment_created",
		v2 + " sop_version_state_changed",
	}
	if strings.Join(order, "\n") != strings.Join(want, "\n") {
		t.Errorf("audit order:\nwant %v\ngot  %v", want, order)
	}
}

func TestLifecycle_EmptyActorIsRefusedAndWritesNothing(t *testing.T) {
	env := newLifecycleEnv(t)
	sopID, vID := versionIn(t, env, sop.StateDraft)
	before := takeSnapshot(t, env, sopID, vID)

	_, _, err := env.SOPService.RegisterSOPVersion(sopID, "# Nobody wrote this", "Anonymous", "")
	if !errors.Is(err, sop.ErrActorRequired) {
		t.Fatalf("create: want ErrActorRequired, got %v", err)
	}
	if err := env.SOPService.PromoteSOPVersion(vID, ""); !errors.Is(err, sop.ErrActorRequired) {
		t.Fatalf("promote: want ErrActorRequired, got %v", err)
	}
	if after := takeSnapshot(t, env, sopID, vID); after != before {
		t.Fatalf("an empty actor changed the record: before %+v, after %+v", before, after)
	}
}

func TestLifecycle_DuplicateReaderAcknowledgmentIsAConflict(t *testing.T) {
	env := newLifecycleEnv(t)
	sopID, vID := versionIn(t, env, sop.StatePublished)
	_, err := env.SOPService.AddAcknowledgment(vID, lifecycleReader, sop.AckTypeRead)
	mustDo(t, "first acknowledgment", err)
	before := takeSnapshot(t, env, sopID, vID)

	_, err = env.SOPService.AddAcknowledgment(vID, lifecycleReader, sop.AckTypeRead)

	if err == nil || err.Error() != "conflict: already acknowledged" {
		t.Fatalf("want conflict: already acknowledged, got %v", err)
	}
	if after := takeSnapshot(t, env, sopID, vID); after != before {
		t.Fatalf("a duplicate acknowledgment changed the record: before %+v, after %+v", before, after)
	}
}

func TestLifecycle_UnknownVersionIsNotFound(t *testing.T) {
	env := newLifecycleEnv(t)
	for name, op := range lifecycleOps {
		if name == "create" {
			continue
		}
		if _, err := op(env, "", "no-such-version"); err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("%s: want not found, got %v", name, err)
		}
	}
}

func TestLifecycle_SavingIdenticalContentWritesNothing(t *testing.T) {
	env := newLifecycleEnv(t)
	sopID, err := env.SOPService.RegisterSOP("Idempotent SOP", strPtr(lifecycleActor))
	mustDo(t, "RegisterSOP", err)
	vID, num, err := env.SOPService.RegisterSOPVersion(sopID, "# Same", "First", lifecycleActor)
	mustDo(t, "first save", err)
	before := takeSnapshot(t, env, sopID, vID)

	againID, againNum, err := env.SOPService.RegisterSOPVersion(sopID, "# Same", "Second", lifecycleActor)

	mustDo(t, "second save", err)
	if againID != vID || againNum != num {
		t.Fatalf("want the existing version %s (%d), got %s (%d)", vID, num, againID, againNum)
	}
	if after := takeSnapshot(t, env, sopID, vID); after != before {
		t.Fatalf("an identical save changed the record: before %+v, after %+v", before, after)
	}
}

func TestLifecycle_FailedCreateRemovesItsContentFile(t *testing.T) {
	env := newLifecycleEnv(t)
	sopID, err := env.SOPService.RegisterSOP("Cleanup SOP", strPtr(lifecycleActor))
	mustDo(t, "RegisterSOP", err)

	// An actor who is not a user fails inside the transaction, after the content file is written.
	_, _, err = env.SOPService.RegisterSOPVersion(sopID, "# Orphan", "First", "not-a-user")

	if err == nil {
		t.Fatal("want an error for an actor who is not a user")
	}
	if n := versionCount(t, env, sopID); n != 0 {
		t.Fatalf("want no version, got %d", n)
	}
	path := filepath.Join(env.DataDir, "sops", sopID, "version-1.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("want the content file removed, stat says %v", err)
	}
}
