package auth

import (
	"database/sql"
	"sync"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// Service orchestrates user, session, and avatar operations.
type Service struct {
	db          *sql.DB
	auditLogger *audit.Logger
	dataDir     string
	// encKey seals TOTP secrets (SECRET_ENCRYPTION_KEY). Without it, 2FA
	// cannot be set up and stored secrets cannot be read.
	encKey []byte
	// mu is the only guard for read-then-write steps that run without a
	// transaction: email-code attempt counting (VerifySecondFactor), session
	// creation (CreateSession), issuing email codes (IssueEmailCode), starting
	// TOTP setup (BeginTOTPSetup), and setting locale and theme (SetUserLocale,
	// SetUserTheme). Keep it until those steps move into transactions, which
	// hold the write lock from begin. It only works within one process.
	mu sync.RWMutex
}

// NewService creates a new auth service. dataDir is the app DATA_DIR used for
// profile picture files under users/<id>/. encKey is the 32-byte
// SECRET_ENCRYPTION_KEY, or nil when it is not configured.
func NewService(db *sql.DB, l *audit.Logger, dataDir string, encKey []byte) *Service {
	return &Service{
		db:          db,
		auditLogger: l,
		dataDir:     dataDir,
		encKey:      encKey,
	}
}

// EncryptionKeyConfigured reports whether 2FA secrets can be sealed and read.
func (s *Service) EncryptionKeyConfigured() bool {
	return len(s.encKey) == 32
}
