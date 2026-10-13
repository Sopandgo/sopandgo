// Package branding stores the organization's name, logo, and primary colour,
// which replace the SOPandGO defaults in the web app and in outgoing messages.
package branding

import "errors"

const (
	// MaxNameRunes caps the name so it fits the navbar and email subjects.
	MaxNameRunes = 40
	// MaxUploadBytes caps a logo upload before it is decoded.
	MaxUploadBytes = 2 << 20
	// maxPixelEdge refuses huge images before their pixels are decoded.
	maxPixelEdge = 4096
	// LogoMaxEdge is the longest side of the stored logo.
	LogoMaxEdge = 512
	// IconSize is the side of the square favicon.
	IconSize = 64

	// Theme backgrounds (base-100 in frontend/src/routes/layout.css). A custom
	// primary must reach MinContrast against the background of its theme.
	LightBase100 = "#ffffff"
	DarkBase100  = "#2c2a29"
	MinContrast  = 4.5
)

// Image variants served by GET /api/branding/logo.
const (
	VariantLogo = "logo"
	VariantIcon = "icon"
)

var (
	ErrNameInvalid    = errors.New("name invalid")
	ErrColourInvalid  = errors.New("colour invalid")
	ErrColourContrast = errors.New("colour contrast too low")
	ErrNoLogo         = errors.New("no logo")
	ErrInvalidVariant = errors.New("invalid variant")
)

// Branding is the public view: every field is safe to show before sign-in.
// Empty strings mean the SOPandGO default.
type Branding struct {
	Name                string `json:"name"`
	HasLogo             bool   `json:"has_logo"`
	LogoHash            string `json:"logo_hash,omitempty"`
	PrimaryLight        string `json:"primary_light,omitempty"`
	PrimaryLightContent string `json:"primary_light_content,omitempty"`
	PrimaryDark         string `json:"primary_dark,omitempty"`
	PrimaryDarkContent  string `json:"primary_dark_content,omitempty"`
}

// TextInput is what an admin saves in one go: name and both theme colours.
// Empty fields reset to the default.
type TextInput struct {
	Name         string `json:"name"`
	PrimaryLight string `json:"primary_light"`
	PrimaryDark  string `json:"primary_dark"`
}
