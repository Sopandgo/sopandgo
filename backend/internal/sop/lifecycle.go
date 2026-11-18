package sop

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// The lifecycle module owns every transition of a version and its side effects.
// It is the only code that opens a transaction for a lifecycle change: the public
// operations (RegisterSOPVersion, PromoteSOPVersion, RejectSOPVersion,
// ApproveSOPVersion, AddAcknowledgment) only request a transition.

type transition string

const (
	transitionCreate    transition = "create"     // new version → draft
	transitionPromote   transition = "promote"    // draft → rc
	transitionReject    transition = "reject"     // rc → rejected
	transitionApprove   transition = "approve"    // rc → published
	transitionReaderAck transition = "reader-ack" // reader acknowledgment on published
)

// transitionRule is one row of the lifecycle table.
type transitionRule struct {
	// createsVersion marks the transition that writes a new version. It needs the
	// request's writeVersion step and names the SOP instead of a version.
	createsVersion bool
	// from lists the states the version may be in. Empty for a new version.
	from []string
	// refusal is the conflict message when the version is in any other state.
	refusal string
	// to is the state written. Empty when the transition writes no state row.
	to string
	// logsStateChange writes a sop_version_state_changed event for the new state.
	// A new version logs sop_version_created instead.
	logsStateChange bool
	// ackType is the acknowledgment written. Empty for none.
	ackType string
	// needsReason requires a reason, stored on the state row and in the audit event.
	needsReason bool
	// pendingRCRefusal refuses the transition while a release candidate is pending for the SOP.
	pendingRCRefusal string
	// supersedesPublished moves the SOP's published version to superseded first.
	supersedesPublished bool
}

// Every pair not listed here is refused, including any transition out of
// published, rejected or superseded, and draft → rejected.
var lifecycleTable = map[transition]transitionRule{
	transitionCreate: {
		createsVersion:   true,
		to:               StateDraft,
		ackType:          AckTypeAuthor,
		pendingRCRefusal: "cannot create a new version while a release candidate is pending approval for this SOP",
	},
	transitionPromote: {
		from:             []string{StateDraft},
		refusal:          "only drafts can be promoted to release candidates",
		to:               StateRC,
		logsStateChange:  true,
		pendingRCRefusal: "a release candidate already exists for this SOP; resolve it before promoting another version",
	},
	transitionReject: {
		from:            []string{StateRC},
		refusal:         "only release candidates can be rejected",
		to:              StateRejected,
		logsStateChange: true,
		needsReason:     true,
	},
	transitionApprove: {
		from:                []string{StateRC},
		refusal:             "only release candidates can be approved",
		to:                  StatePublished,
		logsStateChange:     true,
		ackType:             AckTypeApproved,
		supersedesPublished: true,
	},
	transitionReaderAck: {
		from:    []string{StatePublished},
		refusal: "reader acknowledgments require a published version",
		ackType: AckTypeRead,
	},
}

// newVersionStep writes a new version's content inside the lifecycle transaction.
// It runs after the archived and pending-release-candidate checks and before the
// draft state is written.
type newVersionStep func(tx *sql.Tx) (newVersion, error)

// newVersion is what a newVersionStep wrote.
type newVersion struct {
	id     string
	number int
	// reused is true when the latest version already has this content. Nothing
	// was written and the transition writes nothing else.
	reused bool
	// contentHash is the new version's content hash.
	contentHash string
	// auditPayload is the sop_version_created payload.
	auditPayload map[string]any
}

type transitionRequest struct {
	transition transition
	// versionID names the version for every transition except a new version.
	versionID string
	// sopID names the SOP for a new version.
	sopID       string
	actorUserID string
	// reason is the rejection reason.
	reason string
	// writeVersion is the step that writes a new version.
	writeVersion newVersionStep
}

type transitionResult struct {
	versionID     string
	versionNumber int
	ackID         string
}

// runTransition applies one lifecycle transition in one transaction: it loads the
// version, checks the SOP is not archived, checks the transition against the table,
// writes the state row, acknowledgment and audit events, and commits. The PDF for
// the new state is generated after the commit.
func (s *Service) runTransition(req transitionRequest) (transitionResult, error) {
	rule, ok := lifecycleTable[req.transition]
	if !ok {
		return transitionResult{}, fmt.Errorf("unknown lifecycle transition %q", req.transition)
	}
	if strings.TrimSpace(req.actorUserID) == "" {
		return transitionResult{}, ErrActorRequired
	}
	if rule.createsVersion && req.writeVersion == nil {
		return transitionResult{}, fmt.Errorf("lifecycle transition %q needs a step that writes the version", req.transition)
	}
	var reason *string
	if rule.needsReason {
		r := strings.TrimSpace(req.reason)
		if r == "" {
			return transitionResult{}, ErrRejectReasonRequired
		}
		if utf8.RuneCountInString(r) > MaxRejectReasonRunes {
			return transitionResult{}, ErrRejectReasonTooLong
		}
		reason = &r
	}

	tx, err := s.db.Begin()
	if err != nil {
		return transitionResult{}, err
	}
	defer tx.Rollback()

	// 1. Load the version and its current state.
	sopID := req.sopID
	var current *SOPVersion
	if !rule.createsVersion {
		current, err = getSOPVersionByIDRecord(tx, req.versionID)
		if err != nil {
			return transitionResult{}, err
		}
		sopID = current.SOPID
	}

	// 2. An archived SOP is frozen.
	if err := requireSOPActiveRecord(tx, sopID); err != nil {
		return transitionResult{}, err
	}

	// 3. Check the transition against the table.
	if current != nil && !slices.Contains(rule.from, current.Status) {
		return transitionResult{}, fmt.Errorf("conflict: %s, current state is '%s'", rule.refusal, current.Status)
	}
	if rule.pendingRCRefusal != "" {
		hasRC, err := hasReleaseCandidateForSOPRecord(tx, sopID)
		if err != nil {
			return transitionResult{}, fmt.Errorf("failed to check release candidates: %w", err)
		}
		if hasRC {
			return transitionResult{}, fmt.Errorf("conflict: %s", rule.pendingRCRefusal)
		}
	}

	// 4. Write the state row, the acknowledgment and the audit events.
	actor := req.actorUserID
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result := transitionResult{versionID: req.versionID}
	// contentHash goes into every event that creates or signs the version, so
	// external anchoring of the audit chain also covers the SOP content.
	var contentHash string
	if current != nil {
		result.versionNumber = current.Version
		contentHash = current.ContentHash
	}

	if rule.createsVersion {
		created, err := req.writeVersion(tx)
		if err != nil {
			return transitionResult{}, err
		}
		if created.reused {
			return transitionResult{versionID: created.id, versionNumber: created.number}, nil
		}
		result.versionID = created.id
		result.versionNumber = created.number
		contentHash = created.contentHash
		if err := s.auditLogger.Log(tx, audit.EventSOPVersionCreated, audit.EntitySOPVersion, created.id, &actor, created.auditPayload); err != nil {
			return transitionResult{}, fmt.Errorf("audit log failed: %w", err)
		}
	}

	if rule.supersedesPublished {
		if err := s.supersedePublished(tx, sopID, result.versionID, result.versionNumber, actor, now); err != nil {
			return transitionResult{}, err
		}
	}

	if rule.to != "" {
		if err := createSOPVersionStateWithReasonRecord(tx, uuid.NewString(), result.versionID, rule.to, actor, now, reason); err != nil {
			return transitionResult{}, fmt.Errorf("failed to append %s state: %w", rule.to, err)
		}
	}

	if rule.ackType != "" {
		ackID, err := s.writeAcknowledgment(tx, result.versionID, contentHash, actor, rule.ackType, now)
		if err != nil {
			return transitionResult{}, err
		}
		result.ackID = ackID
	}

	if rule.logsStateChange {
		payload := map[string]any{"new_state": rule.to}
		if reason != nil {
			payload["reason"] = *reason
		}
		if err := s.auditLogger.Log(tx, audit.EventSOPVersionStateChanged, audit.EntitySOPVersion, result.versionID, &actor, payload); err != nil {
			return transitionResult{}, fmt.Errorf("audit log failed for state change: %w", err)
		}
	}

	// 5. Commit, then generate the PDF for the new state.
	if err := tx.Commit(); err != nil {
		return transitionResult{}, err
	}
	if rule.to != "" {
		s.generateVersionPDFArtifactAfterCommit(result.versionID, rule.to, actor)
	}
	return result, nil
}

// supersedePublished moves the SOP's published version, if any, to superseded.
// It runs before the new published state is written, so the SOP never has two.
func (s *Service) supersedePublished(tx *sql.Tx, sopID, publishingVersionID string, publishingVersionNumber int, actor, now string) error {
	oldID, err := getActivePublishedVersionIDRecord(tx, sopID, publishingVersionID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check for older published versions: %w", err)
	}
	if err := createSOPVersionStateRecord(tx, uuid.NewString(), oldID, StateSuperseded, actor, now); err != nil {
		return fmt.Errorf("failed to supersede previous version: %w", err)
	}
	err = s.auditLogger.Log(tx, audit.EventSOPVersionStateChanged, audit.EntitySOPVersion, oldID, &actor, map[string]any{
		"new_state": StateSuperseded,
		"reason":    fmt.Sprintf("Superseded by version %d", publishingVersionNumber),
	})
	if err != nil {
		return fmt.Errorf("audit log failed for superseding old version: %w", err)
	}
	return nil
}

// writeAcknowledgment writes an acknowledgment and its audit event. A reader
// acknowledgment keeps its own event on the acknowledgment entity; author and
// approver acknowledgments are logged on the version.
func (s *Service) writeAcknowledgment(tx *sql.Tx, versionID, contentHash, actor, ackType, now string) (string, error) {
	ackID := uuid.NewString()
	if err := createSOPAcknowledgmentRecord(tx, ackID, versionID, actor, ackType, now); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return "", fmt.Errorf("conflict: already acknowledged")
		}
		return "", fmt.Errorf("failed to record %s acknowledgment: %w", ackType, err)
	}

	var err error
	if ackType == AckTypeRead {
		err = s.auditLogger.Log(tx, audit.EventAcknowledgmentAdded, audit.EntityAcknowledgment, ackID, &actor, map[string]string{
			"sop_version_id": versionID,
			"user_id":        actor,
			"ack_type":       ackType,
			"content_hash":   contentHash,
		})
	} else {
		err = s.auditLogger.Log(tx, audit.EventSOPAcknowledgmentCreated, audit.EntitySOPVersion, versionID, &actor, map[string]any{
			"ack_type":     ackType,
			"ack_id":       ackID,
			"content_hash": contentHash,
		})
	}
	if err != nil {
		return "", fmt.Errorf("audit log failed for %s acknowledgment: %w", ackType, err)
	}
	return ackID, nil
}
