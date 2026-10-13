package branding

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/i18n"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func newTestService(t *testing.T) (*Service, *audit.Logger) {
	t.Helper()
	store, err := storage.OpenInMemory()
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	t.Cleanup(func() { store.DB.Close() })
	// Audit events reference their actor, so the tests act as a real user.
	if _, err := store.DB.Exec(`
		INSERT INTO users (id, display_name, email, password_hash, role_id, is_active, created_at)
		VALUES ('admin', 'Admin', 'admin@test.local', 'x', 'admin', 1, '2026-01-01T00:00:00Z')
	`); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	logger, err := audit.New(store.DB)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	t.Cleanup(func() { i18n.SetAppName("") })
	return NewService(store.DB, logger), logger
}

func testImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 20, G: 80, B: 200, A: 255})
		}
	}
	return img
}

func encode(t *testing.T, img image.Image, format string) []byte {
	t.Helper()
	var buf bytes.Buffer
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&buf, img, nil)
	} else {
		err = png.Encode(&buf, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestNormalizeName(t *testing.T) {
	if got, err := NormalizeName("  Lab Handbook  "); err != nil || got != "Lab Handbook" {
		t.Fatalf("trim: %q %v", got, err)
	}
	if got, err := NormalizeName(strings.Repeat("ä", MaxNameRunes)); err != nil || got == "" {
		t.Fatalf("40 runes should pass: %v", err)
	}
	for _, product := range []string{"Sopandgo", "SOP and GO", "sopandgo", "SOP & GO", "SOPandGO"} {
		if got, err := NormalizeName(product); err != nil || got != "" {
			t.Fatalf("%q should mean the default name, got %q %v", product, got, err)
		}
	}
	for _, bad := range []string{strings.Repeat("a", MaxNameRunes+1), "Lab\r\nBcc: x@y", "tab\there"} {
		if _, err := NormalizeName(bad); !errors.Is(err, ErrNameInvalid) {
			t.Fatalf("%q: want ErrNameInvalid, got %v", bad, err)
		}
	}
}

func TestNormalizePrimary(t *testing.T) {
	cases := []struct {
		colour, base string
		want         error
	}{
		{"", LightBase100, nil},
		{"#1D4ED8", LightBase100, nil},               // blue-700 on white ≈ 6.7:1
		{"#facc15", LightBase100, ErrColourContrast}, // yellow on white
		{"#facc15", DarkBase100, nil},                // yellow on ink
		{"#1d4ed8", DarkBase100, ErrColourContrast},
		{"1d4ed8", LightBase100, ErrColourInvalid},
		{"#1d4ed", LightBase100, ErrColourInvalid},
		{"red", LightBase100, ErrColourInvalid},
		{"#1d4ed8;}body{", LightBase100, ErrColourInvalid},
	}
	for _, c := range cases {
		value, content, err := NormalizePrimary(c.colour, c.base)
		if !errors.Is(err, c.want) {
			t.Fatalf("%q on %s: want %v, got %v", c.colour, c.base, c.want, err)
		}
		if err == nil && c.colour != "" {
			if value != strings.ToLower(c.colour) || content != c.base {
				t.Fatalf("%q: got value %q content %q", c.colour, value, content)
			}
		}
	}
	// The bar is inclusive: #767676 on white is the classic 4.54:1 grey.
	if _, _, err := NormalizePrimary("#767676", LightBase100); err != nil {
		t.Fatalf("#767676 should pass: %v", err)
	}
	if _, _, err := NormalizePrimary("#777777", LightBase100); !errors.Is(err, ErrColourContrast) {
		t.Fatalf("#777777 should fail: %v", err)
	}
}

func TestProcessLogo(t *testing.T) {
	t.Run("wide PNG keeps aspect, icon is square", func(t *testing.T) {
		logo, icon, err := processLogo(encode(t, testImage(1200, 300), "png"))
		if err != nil {
			t.Fatal(err)
		}
		l, err := png.Decode(bytes.NewReader(logo))
		if err != nil {
			t.Fatal(err)
		}
		if l.Bounds().Dx() != LogoMaxEdge || l.Bounds().Dy() != 128 {
			t.Fatalf("logo size %v", l.Bounds())
		}
		i, err := png.Decode(bytes.NewReader(icon))
		if err != nil {
			t.Fatal(err)
		}
		if i.Bounds().Dx() != IconSize || i.Bounds().Dy() != IconSize {
			t.Fatalf("icon size %v", i.Bounds())
		}
		if _, _, _, a := i.At(0, 0).RGBA(); a != 0 {
			t.Fatalf("icon padding should be transparent, alpha %d", a)
		}
	})
	t.Run("small JPEG is not upscaled", func(t *testing.T) {
		logo, _, err := processLogo(encode(t, testImage(100, 40), "jpeg"))
		if err != nil {
			t.Fatal(err)
		}
		l, _ := png.Decode(bytes.NewReader(logo))
		if l.Bounds().Dx() != 100 || l.Bounds().Dy() != 40 {
			t.Fatalf("logo size %v", l.Bounds())
		}
	})
	t.Run("rejects SVG and text", func(t *testing.T) {
		for _, raw := range [][]byte{
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
			[]byte("hello"),
			nil,
		} {
			if _, _, err := processLogo(raw); err == nil {
				t.Fatalf("%q should be rejected", raw)
			}
		}
	})
	t.Run("rejects oversized dimensions", func(t *testing.T) {
		_, _, err := processLogo(encode(t, testImage(maxPixelEdge+1, 1), "png"))
		if err == nil || !strings.Contains(err.Error(), "image too large") {
			t.Fatalf("want image too large, got %v", err)
		}
	})
}

func TestServiceLifecycle(t *testing.T) {
	svc, logger := newTestService(t)

	b, err := svc.Get()
	if err != nil || b != (Branding{}) {
		t.Fatalf("defaults: %+v %v", b, err)
	}
	if _, _, err := svc.Image(VariantLogo); !errors.Is(err, ErrNoLogo) {
		t.Fatalf("no logo yet: %v", err)
	}
	if _, err := svc.RemoveLogo("admin"); !errors.Is(err, ErrNoLogo) {
		t.Fatalf("remove without logo: %v", err)
	}

	b, err = svc.UpdateText(TextInput{Name: " Lab Handbook ", PrimaryLight: "#1D4ED8", PrimaryDark: "#93c5fd"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "Lab Handbook" || b.PrimaryLight != "#1d4ed8" || b.PrimaryLightContent != LightBase100 ||
		b.PrimaryDark != "#93c5fd" || b.PrimaryDarkContent != DarkBase100 {
		t.Fatalf("after update: %+v", b)
	}
	if i18n.AppName() != "Lab Handbook" {
		t.Fatalf("i18n name not updated: %q", i18n.AppName())
	}

	if _, err := svc.UpdateText(TextInput{PrimaryLight: "#ffff00"}, "admin"); !errors.Is(err, ErrColourContrast) {
		t.Fatalf("low contrast should fail: %v", err)
	}
	if b, _ := svc.Get(); b.Name != "Lab Handbook" {
		t.Fatalf("failed update must not change anything: %+v", b)
	}

	b, err = svc.SetLogo(encode(t, testImage(300, 100), "png"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if !b.HasLogo || b.LogoHash == "" || b.Name != "Lab Handbook" {
		t.Fatalf("after logo: %+v", b)
	}
	if data, hash, err := svc.Image(VariantIcon); err != nil || len(data) == 0 || hash != b.LogoHash {
		t.Fatalf("icon: %d bytes, %q, %v", len(data), hash, err)
	}
	if _, _, err := svc.Image("../etc"); !errors.Is(err, ErrInvalidVariant) {
		t.Fatalf("bad variant: %v", err)
	}

	b, err = svc.UpdateText(TextInput{}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "" || b.PrimaryLight != "" || !b.HasLogo {
		t.Fatalf("text reset must keep the logo: %+v", b)
	}
	if i18n.AppName() != i18n.DefaultAppName {
		t.Fatalf("i18n name not reset: %q", i18n.AppName())
	}

	b, err = svc.RemoveLogo("admin")
	if err != nil || b.HasLogo || b.LogoHash != "" {
		t.Fatalf("after remove: %+v %v", b, err)
	}

	report, err := logger.VerifyEntireChain()
	if err != nil || report.Invalid != 0 {
		t.Fatalf("audit chain: %+v %v", report, err)
	}
	events, total, err := logger.List(50, 0, audit.ListFilters{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, ev := range events {
		seen[ev.EventType]++
	}
	if seen[audit.EventBrandingUpdated] != 2 || seen[audit.EventBrandingLogoUpdated] != 1 || seen[audit.EventBrandingLogoRemoved] != 1 {
		t.Fatalf("audit events (%d total): %v", total, seen)
	}
}

func TestLoadSetsAppName(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.UpdateText(TextInput{Name: "Restored Lab"}, "admin"); err != nil {
		t.Fatal(err)
	}
	i18n.SetAppName("")
	if err := svc.Load(); err != nil {
		t.Fatal(err)
	}
	if i18n.AppName() != "Restored Lab" {
		t.Fatalf("Load: %q", i18n.AppName())
	}
}

func TestResetRestoresEveryDefault(t *testing.T) {
	svc, logger := newTestService(t)

	// Nothing set: a no-op, not an error, and nothing audited.
	if b, err := svc.Reset("admin"); err != nil || b != (Branding{}) {
		t.Fatalf("reset on defaults: %+v %v", b, err)
	}

	if _, err := svc.UpdateText(TextInput{Name: "Lab Handbook", PrimaryLight: "#1d4ed8", PrimaryDark: "#93c5fd"}, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetLogo(encode(t, testImage(120, 40), "png"), "admin"); err != nil {
		t.Fatal(err)
	}

	b, err := svc.Reset("admin")
	if err != nil || b != (Branding{}) {
		t.Fatalf("after reset: %+v %v", b, err)
	}
	if _, _, err := svc.Image(VariantLogo); !errors.Is(err, ErrNoLogo) {
		t.Fatalf("logo should be gone: %v", err)
	}
	if i18n.AppName() != i18n.DefaultAppName {
		t.Fatalf("i18n name not reset: %q", i18n.AppName())
	}

	events, _, err := logger.List(50, 0, audit.ListFilters{EventType: audit.EventBrandingReset})
	if err != nil || len(events) != 1 {
		t.Fatalf("want one branding_reset event, got %d (%v)", len(events), err)
	}
	if report, err := logger.VerifyEntireChain(); err != nil || report.Invalid != 0 {
		t.Fatalf("audit chain: %+v %v", report, err)
	}
}

func TestStoredProductNameReadsAsDefault(t *testing.T) {
	svc, _ := newTestService(t)
	// A name saved before the rule existed.
	if _, err := svc.db.Exec(`INSERT INTO branding (id, name, updated_at) VALUES (1, 'Sopandgo', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if b, err := svc.Get(); err != nil || b.Name != "" {
		t.Fatalf("stored product spelling should read as default: %+v %v", b, err)
	}
	if err := svc.Load(); err != nil || i18n.AppName() != i18n.DefaultAppName {
		t.Fatalf("Load: %q %v", i18n.AppName(), err)
	}
}
