package audit

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Hash versions stored in audit_events.hash_version.
//
// Version 1 rows were written before migration v30. Their hash concatenates
// fields with no separators and does not cover entity_type or actor_user_id.
// Version 2 is written for every new row. It is the algorithm that external
// anchoring and the auditor script in docs/ops/verifying-anchors.md rely on.
const (
	HashV1 = 1
	HashV2 = 2

	// CurrentHashVersion is the version new rows are written with.
	CurrentHashVersion = HashV2
)

// hashV2Domain separates v2 row hashes from any other SHA-256 input.
const hashV2Domain = "sopandgo-audit-v2"

// HashInput is every column that a row hash can cover.
type HashInput struct {
	PrevHash    string
	EventType   string
	EntityType  string
	EntityID    string
	ActorUserID *string
	Payload     string
	CreatedAt   string // exactly as stored (RFC 3339 with nanoseconds, UTC)
}

// ComputeHash returns the lowercase hex row hash for the given version.
func ComputeHash(version int, in HashInput) (string, error) {
	switch version {
	case HashV1:
		return hashV1(in), nil
	case HashV2:
		return hashV2(in), nil
	default:
		return "", fmt.Errorf("unknown audit hash version %d", version)
	}
}

// hashV1 is SHA256(prev + event_type + entity_id + payload + created_at).
func hashV1(in HashInput) string {
	sum := sha256.Sum256([]byte(in.PrevHash + in.EventType + in.EntityID + in.Payload + in.CreatedAt))
	return hex.EncodeToString(sum[:])
}

// hashV2 is SHA256 over the domain tag followed by each field, each written
// as a 4-byte big-endian length and then its UTF-8 bytes, in this order:
// prev_hash, event_type, entity_type, entity_id, actor_user_id, payload,
// created_at. A NULL actor_user_id is written as the empty string; user ids
// are never empty.
func hashV2(in HashInput) string {
	actor := ""
	if in.ActorUserID != nil {
		actor = *in.ActorUserID
	}
	h := sha256.New()
	h.Write([]byte(hashV2Domain))
	var lenBuf [4]byte
	for _, field := range []string{in.PrevHash, in.EventType, in.EntityType, in.EntityID, actor, in.Payload, in.CreatedAt} {
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(field)))
		h.Write(lenBuf[:])
		h.Write([]byte(field))
	}
	return hex.EncodeToString(h.Sum(nil))
}
