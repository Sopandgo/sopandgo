package pdfgen

import (
	"strings"
	"testing"
	"time"
)

func TestHTMLDocument_GermanChrome(t *testing.T) {
	html, err := HTMLDocument(Input{
		Markdown: "# Protocol\n\nRinse the gel.",
		Meta: Metadata{
			SOPTitle:         "Water Protocol",
			SOPID:            "sop-1",
			SOPVersionID:     "ver-1",
			SOPVersionNumber: 2,
			Stage:            "published",
			ContentHash:      "abc",
			GeneratorVersion: "test",
			GeneratedAt:      time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC),
			Locale:           "de",
		},
	})
	if err != nil {
		t.Fatalf("html: %v", err)
	}
	footer, err := FooterHTML(Input{
		Meta: Metadata{Locale: "de", Stage: "published", GeneratedAt: time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("footer: %v", err)
	}
	if !strings.Contains(footer, "Vertraulich – nur für den internen Gebrauch") {
		t.Fatalf("missing German confidentiality label:\n%s", footer)
	}
	if !strings.Contains(html, "Lebenszyklusstatus") || !strings.Contains(html, "Veröffentlicht") {
		t.Fatalf("missing German stage chrome:\n%s", html)
	}
	if !strings.Contains(html, "Water Protocol") || !strings.Contains(html, "Rinse the gel.") {
		t.Fatalf("authored SOP text should stay verbatim:\n%s", html)
	}
}
