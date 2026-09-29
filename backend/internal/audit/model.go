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
}

type ChainFailure struct {
	EventID int64  `json:"event_id"`
	Error   string `json:"error"`
}
