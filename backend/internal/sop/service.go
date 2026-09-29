package sop

import (
	"database/sql"
	"sync"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/pdfgen"
)

// Service orchestrates SOP operations, managing state and concurrency.
type Service struct {
	db                  *sql.DB
	dataDir             string
	auditLogger         *audit.Logger
	pdfGenerator        pdfgen.Renderer
	pdfGeneratorVersion string
	pdfExportEnabled    bool
	mu                  sync.RWMutex
}

// NewService creates a new SOP service instance.
func NewService(db *sql.DB, l *audit.Logger, dataDir string, pdfGeneratorVersion string, pdfExportEnabled bool, renderer pdfgen.Renderer) *Service {
	return &Service{
		db:                  db,
		auditLogger:         l,
		dataDir:             dataDir,
		pdfGenerator:        renderer,
		pdfGeneratorVersion: pdfGeneratorVersion,
		pdfExportEnabled:    pdfExportEnabled,
	}
}

func (s *Service) IsPDFExportEnabled() bool {
	return s.pdfExportEnabled && s.pdfGenerator != nil
}
