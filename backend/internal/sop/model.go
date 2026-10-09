package sop

import (
	"errors"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

const MaxChangeSummaryRunes = 500

var (
	ErrChangeSummaryRequired = errors.New("change summary is required")
	ErrChangeSummaryTooLong  = errors.New("change summary must be 500 characters or fewer")
)

type SOP struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type SOPWithTags struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	Tags       []Tag     `json:"tags"`
	IsFavorite bool      `json:"is_favorite"`
}

type SOPListItem struct {
	SOP
	Tags       []Tag `json:"tags"`
	IsFavorite bool  `json:"is_favorite"`
	// LatestVersion is the highest version number in its current state; nil before the first version.
	LatestVersion *SOPListVersion `json:"latest_version"`
	// PublishedVersion is the number of the version readers see; nil until one is published.
	PublishedVersion *int `json:"published_version"`
}

type SOPListVersion struct {
	Version   int       `json:"version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	IsActive bool   `json:"is_active"`
}

type SOPVersion struct {
	ID            string    `json:"id"`
	SOPID         string    `json:"sop_id"`
	Version       int       `json:"version"`
	ContentPath   string    `json:"content_path"`
	ContentHash   string    `json:"content_hash"`
	ChangeSummary string    `json:"change_summary"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
}

// SOP version states
const (
	StateDraft      = "draft"
	StateRC         = "rc"
	StatePublished  = "published"
	StateRejected   = "rejected"
	StateSuperseded = "superseded"
)

type SOPVersionState struct {
	ID           string    `json:"id"`
	SOPVersionID string    `json:"sop_version_id"`
	State        string    `json:"state"`
	ActorUserID  string    `json:"actor_user_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type SOPVersionSummary struct {
	ID              string                      `json:"id"`
	SOPID           string                      `json:"sop_id"`
	Version         int                         `json:"version"`
	Status          string                      `json:"status"`
	ChangeSummary   string                      `json:"change_summary"`
	Content         string                      `json:"content"`
	ContentHash     string                      `json:"content_hash"`
	Assets          []SOPAsset                  `json:"assets"`
	Acknowledgments []SOPAcknowledgmentWithUser `json:"acknowledgments"`
	CreatedAt       time.Time                   `json:"created_at"`
	HashValid       bool                        `json:"hash_valid"`
	Tags            []Tag                       `json:"tags"`
}

type SOPVersionPDFArtifact struct {
	ID               string    `json:"id"`
	SOPID            string    `json:"sop_id"`
	SOPVersionID     string    `json:"sop_version_id"`
	Stage            string    `json:"stage"`
	GeneratorVersion string    `json:"generator_version"`
	FilePath         string    `json:"file_path"`
	ContentHash      string    `json:"content_hash"`
	SizeBytes        int64     `json:"size_bytes"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
}

type SOPAsset struct {
	ID          string    `json:"id"`
	SOPID       string    `json:"sop_id"`
	Filename    string    `json:"file_name"`
	ContentPath string    `json:"content_path"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

type SOPAcknowledgment struct {
	ID                 string    `json:"id"`
	SOPVersionID       string    `json:"sop_version_id"`
	UserID             string    `json:"user_id"`
	AcknowledgmentType string    `json:"ack_type"`
	CreatedAt          time.Time `json:"created_at"`
}

type SOPAcknowledgmentWithUser struct {
	ID                 string    `json:"id"`
	SOPVersionID       string    `json:"sop_version_id"`
	UserID             string    `json:"user_id"`
	AcknowledgmentType string    `json:"ack_type"`
	CreatedAt          time.Time `json:"created_at"`
	User               auth.User `json:"user"`
}

// Acknowledgment Types
const (
	AckTypeAuthor   = "author"
	AckTypeApproved = "approver"
	AckTypeRead     = "reader"
)

// Integrity

type SystemIntegrityReport struct {
	OK        bool             `json:"ok"`
	CheckedAt string           `json:"checked_at"`
	Versions  IntegritySection `json:"versions"`
	Assets    IntegritySection `json:"assets"`
}

type IntegritySection struct {
	Checked int                `json:"checked"`
	Valid   int                `json:"valid"`
	Invalid int                `json:"invalid"`
	Failed  []IntegrityFailure `json:"failed,omitempty"` // Changed from map to struct
}

type IntegrityFailure struct {
	SOPID   string `json:"sop_id"`
	Version int    `json:"version,omitempty"`  // Only for version failures
	AssetID string `json:"asset_id,omitempty"` // Only for asset failures
	Error   string `json:"error"`
}

// User Signature Tracking

type DiffLine struct {
	Kind string `json:"kind"` // context, add, del
	Text string `json:"text"`
}

type VersionDiff struct {
	Comparable    bool       `json:"comparable"`
	FromVersionID string     `json:"from_version_id,omitempty"`
	FromVersion   int        `json:"from_version,omitempty"`
	ToVersionID   string     `json:"to_version_id"`
	ToVersion     int        `json:"to_version"`
	Lines         []DiffLine `json:"lines"`
}

type PublishedActivity struct {
	SOPID         string    `json:"sop_id"`
	Title         string    `json:"title"`
	VersionID     string    `json:"version_id"`
	Version       int       `json:"version"`
	ChangeSummary string    `json:"change_summary"`
	PublishedAt   time.Time `json:"published_at"`
	PublishedBy   string    `json:"published_by"`
}

type TrainingMember struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	SignedAt    *string `json:"signed_at,omitempty"`
}

type SOPTrainingCoverage struct {
	SOPID        string           `json:"sop_id"`
	Title        string           `json:"title"`
	HasPublished bool             `json:"has_published"`
	VersionID    string           `json:"version_id,omitempty"`
	Version      int              `json:"version,omitempty"`
	Signed       []TrainingMember `json:"signed"`
	Unsigned     []TrainingMember `json:"unsigned"`
}

type UserSignatureStatus struct {
	SOPID              string  `json:"sop_id"`
	Title              string  `json:"title"`
	LatestVersionID    string  `json:"latest_version_id"`
	LatestVersion      int     `json:"latest_version"`
	HasSignedLatest    bool    `json:"has_signed_latest"`
	SignedOlderVersion bool    `json:"signed_older_version"`
	LastSignDate       *string `json:"last_sign_date,omitempty"` // Pointer to handle nulls
}
