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
	mu          sync.RWMutex
}

// NewService creates a new auth service. dataDir is the app DATA_DIR used for
// profile picture files under users/<id>/.
func NewService(db *sql.DB, l *audit.Logger, dataDir string) *Service {
	return &Service{
		db:          db,
		auditLogger: l,
		dataDir:     dataDir,
	}
}
