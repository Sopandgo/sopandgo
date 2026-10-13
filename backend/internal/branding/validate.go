package branding

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var hexColour = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// productName matches any spelling of the product name: split, joined, any case.
var productName = regexp.MustCompile(`(?i)^sop\s*(?:and|&|\+)\s*go$`)

// NormalizeName trims the name and rejects control characters (which would
// also allow header injection in email subjects) and overly long names.
// An empty result means the default name. Any spelling of the product name
// also means the default, so it always reads "SOPandGO".
func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", ErrNameInvalid
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return "", ErrNameInvalid
		}
	}
	if productName.MatchString(name) {
		return "", nil
	}
	return name, nil
}

// NormalizePrimary checks a '#rrggbb' colour against the theme background and
// returns it in lower case with its content colour. Text on the primary
// (buttons) uses the theme background: contrast is symmetric, so it passes too.
// An empty colour means the theme default and returns two empty strings.
func NormalizePrimary(colour, base100 string) (value, content string, err error) {
	colour = strings.TrimSpace(colour)
	if colour == "" {
		return "", "", nil
	}
	if !hexColour.MatchString(colour) {
		return "", "", ErrColourInvalid
	}
	colour = strings.ToLower(colour)
	if ContrastRatio(colour, base100) < MinContrast {
		return "", "", ErrColourContrast
	}
	return colour, base100, nil
}

// ContrastRatio is the WCAG 2 contrast ratio of two '#rrggbb' colours.
func ContrastRatio(a, b string) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(hex string) float64 {
	channel := func(i int) float64 {
		v, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(1) + 0.7152*channel(3) + 0.0722*channel(5)
}
