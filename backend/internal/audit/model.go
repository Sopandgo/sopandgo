package audit

import "time"

type AuditEvent struct {
	ID         int64     `json:"id"`
	EventType  string    `json:"event_type"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	ActorID    *string   `json:"actor_user_id"`
	ActorName  *string   `json:"actor_name"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
	Hash       string    `json:"hash"`
	PrevHash   string    `json:"prev_hash"`
	// HashVersion selects the row hash algorithm (see hash.go).
	HashVersion int `json:"hash_version"`

	// createdAtRaw is created_at exactly as stored, which is what the hash
	// covers. CreatedAt is the parsed value for display.
	createdAtRaw string
}

// CreatedAtRaw returns created_at exactly as stored and hashed.
func (e AuditEvent) CreatedAtRaw() string { return e.createdAtRaw }

// hashInput returns the fields the row hash covers.
func (e AuditEvent) hashInput() HashInput {
	return HashInput{
		PrevHash:    e.PrevHash,
		EventType:   e.EventType,
		EntityType:  e.EntityType,
		EntityID:    e.EntityID,
		ActorUserID: e.ActorID,
		Payload:     e.Payload,
		CreatedAt:   e.createdAtRaw,
	}
}

// rowHashValid recomputes the row hash and compares it with the stored one.
func (e AuditEvent) rowHashValid() bool {
	calc, err := ComputeHash(e.HashVersion, e.hashInput())
	return err == nil && calc == e.Hash
}

type ListFilters struct {
	EventType   string
	EntityType  string
	ActorUserID string
	SOPID       string
}

type FilterOptions struct {
	EventTypes  []string `json:"event_types"`
	EntityTypes []string `json:"entity_types"`
}

// The API response type
type AuditEventWithVerification struct {
	AuditEvent
	HashValid bool `json:"hash_valid"`
}

type ChainIntegrityReport struct {
	Checked int
	Valid   int
	Invalid int
	Failed  []ChainFailure
	// TipID and TipHash are the last row checked (0 and GenesisHash when the
	// range was empty).
	TipID   int64
	TipHash string
}

type ChainFailure struct {
	EventID int64  `json:"event_id"`
	Error   string `json:"error"`
}
