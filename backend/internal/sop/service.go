package sop

import (
	"database/sql"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/pdfgen"
)

// Service orchestrates SOP operations.
//
// Consistency comes from transactions: the storage package opens every
// transaction with the write lock held from begin, so concurrent writers
// serialize at Begin and no process-wide lock is needed. Reads use plain
// queries and never wait behind writes or PDF generation.
type Service struct {
	db                  *sql.DB
	dataDir             string
	auditLogger         *audit.Logger
	pdfGenerator        pdfgen.Renderer
	pdfGeneratorVersion string
	pdfExportEnabled    bool
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
