package i18n

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestSupportedLocalesMatchRepoList(t *testing.T) {
	repo, err := os.ReadFile("../../../i18n/supported-locales.json")
	if err != nil {
		t.Fatalf("read repo list: %v", err)
	}
	var fromRepo []string
	if err := json.Unmarshal(repo, &fromRepo); err != nil {
		t.Fatalf("parse repo list: %v", err)
	}
	if !reflect.DeepEqual(Supported(), fromRepo) {
		t.Fatalf("embedded locales %v != i18n/supported-locales.json %v", Supported(), fromRepo)
	}
}

func TestFallbackAndMissingKey(t *testing.T) {
	if got := Fallback("nope"); got != Base {
		t.Fatalf("unknown locale: got %q", got)
	}
	if got := Fallback(" DE "); got != "de" {
		t.Fatalf("normalize: got %q", got)
	}
	if _, ok := Normalize("fr"); ok {
		t.Fatal("fr should not be supported yet")
	}
	got := T("de", "mail.published.no_summary", nil)
	if got != "Es wurde keine Änderungszusammenfassung erfasst." {
		t.Fatalf("german summary fallback: %q", got)
	}
	missing := T("de", "does.not.exist", nil)
	if missing != "does.not.exist" {
		t.Fatalf("missing key: %q", missing)
	}
	title := T("zz", "notify.sop_published.title", map[string]string{"title": "Water Protocol"})
	if title != "SOP published: Water Protocol" {
		t.Fatalf("unknown locale should use English chrome, got %q", title)
	}
	deTitle := T("de", "notify.sop_published.title", map[string]string{"title": "Water Protocol"})
	if deTitle != "SOP veröffentlicht: Water Protocol" {
		t.Fatalf("german chrome should keep the SOP title, got %q", deTitle)
	}
}
