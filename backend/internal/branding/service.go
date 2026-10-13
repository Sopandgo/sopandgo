package branding

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/integrity"
)

// auditEntityID names the single branding row in audit events.
const auditEntityID = "branding"

// Service reads and changes the organization's branding. Every change is
// audited in the same transaction and updates the name used by i18n.T.
type Service struct {
	db          *sql.DB
	auditLogger *audit.Logger
	mu          sync.Mutex
}

func NewService(db *sql.DB, auditLogger *audit.Logger) *Service {
	return &Service{db: db, auditLogger: auditLogger}
}

// Load reads the stored name into i18n so emails and notifications use it.
// Call it once at startup; a restored backup is applied before that.
func (s *Service) Load() error {
	r, err := getRecord(s.db)
	if err != nil {
		return err
	}
	i18n.SetAppName(customName(r.name))
	return nil
}

// Get returns the public branding.
func (s *Service) Get() (Branding, error) {
	r, err := getRecord(s.db)
	if err != nil {
		return Branding{}, err
	}
	return toBranding(r), nil
}

func toBranding(r record) Branding {
	b := Branding{Name: customName(r.name), HasLogo: r.hasLogo}
	if r.hasLogo {
		b.LogoHash = r.hash
	}
	// Stored colours passed NormalizePrimary; deriving content here keeps one source of truth.
	if light, content, err := NormalizePrimary(r.light, LightBase100); err == nil {
		b.PrimaryLight, b.PrimaryLightContent = light, content
	}
	if dark, content, err := NormalizePrimary(r.dark, DarkBase100); err == nil {
		b.PrimaryDark, b.PrimaryDarkContent = dark, content
	}
	return b
}

// UpdateText validates and saves the name and both theme colours.
func (s *Service) UpdateText(in TextInput, actorID string) (Branding, error) {
	name, err := NormalizeName(in.Name)
	if err != nil {
		return Branding{}, err
	}
	light, _, err := NormalizePrimary(in.PrimaryLight, LightBase100)
	if err != nil {
		return Branding{}, err
	}
	dark, _, err := NormalizePrimary(in.PrimaryDark, DarkBase100)
	if err != nil {
		return Branding{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return Branding{}, err
	}
	defer tx.Rollback()

	before, err := getRecord(tx)
	if err != nil {
		return Branding{}, err
	}
	if err := setTextRecord(tx, name, light, dark, time.Now()); err != nil {
		return Branding{}, err
	}
	if err := s.auditLogger.Log(tx, audit.EventBrandingUpdated, audit.EntitySystem, auditEntityID, &actorID,
		map[string]string{
			"old_name":          before.name,
			"new_name":          name,
			"old_primary_light": before.light,
			"new_primary_light": light,
			"old_primary_dark":  before.dark,
			"new_primary_dark":  dark,
		}); err != nil {
		return Branding{}, err
	}
	return s.commit(tx)
}

// SetLogo processes an uploaded image and stores it as the logo and favicon.
func (s *Service) SetLogo(raw []byte, actorID string) (Branding, error) {
	logo, icon, err := processLogo(raw)
	if err != nil {
		return Branding{}, err
	}
	hash := integrity.HashBytes(logo)

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return Branding{}, err
	}
	defer tx.Rollback()

	if err := setLogoRecord(tx, logo, icon, hash, time.Now()); err != nil {
		return Branding{}, err
	}
	if err := s.auditLogger.Log(tx, audit.EventBrandingLogoUpdated, audit.EntitySystem, auditEntityID, &actorID,
		map[string]string{"content_hash": hash}); err != nil {
		return Branding{}, err
	}
	return s.commit(tx)
}

// RemoveLogo restores the default logo. ErrNoLogo when none is set.
func (s *Service) RemoveLogo(actorID string) (Branding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return Branding{}, err
	}
	defer tx.Rollback()

	before, err := getRecord(tx)
	if err != nil {
		return Branding{}, err
	}
	if !before.hasLogo {
		return Branding{}, ErrNoLogo
	}
	if err := clearLogoRecord(tx, time.Now()); err != nil {
		return Branding{}, err
	}
	if err := s.auditLogger.Log(tx, audit.EventBrandingLogoRemoved, audit.EntitySystem, auditEntityID, &actorID,
		map[string]string{"content_hash": before.hash}); err != nil {
		return Branding{}, err
	}
	return s.commit(tx)
}

// Reset restores every default (name, colours, logo) in one step. Nothing to
// reset is not an error and writes no audit event.
func (s *Service) Reset(actorID string) (Branding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return Branding{}, err
	}
	defer tx.Rollback()

	before, err := getRecord(tx)
	if err != nil {
		return Branding{}, err
	}
	if before == (record{}) {
		return s.commit(tx)
	}
	if err := deleteRecord(tx); err != nil {
		return Branding{}, err
	}
	if err := s.auditLogger.Log(tx, audit.EventBrandingReset, audit.EntitySystem, auditEntityID, &actorID,
		map[string]string{
			"old_name":          before.name,
			"old_primary_light": before.light,
			"old_primary_dark":  before.dark,
			"old_logo_hash":     before.hash,
		}); err != nil {
		return Branding{}, err
	}
	return s.commit(tx)
}

// Image returns the PNG for a variant (logo or icon) and its content hash.
func (s *Service) Image(variant string) ([]byte, string, error) {
	if variant != VariantLogo && variant != VariantIcon {
		return nil, "", ErrInvalidVariant
	}
	return getImageRecord(s.db, variant)
}

// commit finishes a change and returns the new branding. Reading inside the
// transaction keeps the result and the i18n name in step with what was saved.
func (s *Service) commit(tx *sql.Tx) (Branding, error) {
	r, err := getRecord(tx)
	if err != nil {
		return Branding{}, err
	}
	if err := tx.Commit(); err != nil {
		return Branding{}, err
	}
	i18n.SetAppName(customName(r.name))
	return toBranding(r), nil
}

// customName is the stored name, or "" (the default) when it is a spelling of
// the product name saved before NormalizeName mapped those to the default.
func customName(stored string) string {
	if productName.MatchString(strings.TrimSpace(stored)) {
		return ""
	}
	return stored
}
