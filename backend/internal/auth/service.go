package auth

import (
	"database/sql"
	"sync"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// Service orchestrates SOP operations, managing state and concurrency.
type Service struct {
	db          *sql.DB
	auditLogger *audit.Logger
	mu          sync.RWMutex
}

// NewService creates a new SOP service instance.
func NewService(db *sql.DB, l *audit.Logger) *Service {
	return &Service{
		db:          db,
		auditLogger: l,
	}
}
