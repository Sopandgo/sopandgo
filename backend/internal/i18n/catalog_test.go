package i18n

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
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
	if _, ok := Normalize("xx"); ok {
		t.Fatal("xx should not be supported")
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

func TestAppNameFillsMessages(t *testing.T) {
	t.Cleanup(func() { SetAppName("") })

	if got := T("en", "mail.test.subject", nil); got != "SOPandGO email test" {
		t.Fatalf("default name: %q", got)
	}
	SetAppName("  Lab Handbook ")
	if got := T("de", "mail.reset.subject", nil); !strings.Contains(got, "Lab Handbook") {
		t.Fatalf("custom name missing: %q", got)
	}
	if got := T("en", "mail.test.subject", map[string]string{"app_name": "Override"}); got != "Override email test" {
		t.Fatalf("explicit arg should win: %q", got)
	}
	SetAppName("")
	if got := AppName(); got != DefaultAppName {
		t.Fatalf("blank should restore default, got %q", got)
	}
}

func TestPlaceholdersAreReplacedOnce(t *testing.T) {
	t.Cleanup(func() { SetAppName("") })
	SetAppName("{url}")
	got := T("en", "mail.invite.body", map[string]string{"name": "{app_name}", "url": "https://x"})
	if !strings.Contains(got, "Hi {app_name},") || !strings.Contains(got, "Your {url} user account") {
		t.Fatalf("values must not be expanded again: %q", got)
	}
}

func TestNoShippedMessageHardCodesProductName(t *testing.T) {
	load()
	for tag, msgs := range bundles {
		for key, msg := range msgs {
			if strings.Contains(msg, "SOPandGO") {
				t.Errorf("%s %s: use {app_name} instead of the product name", tag, key)
			}
		}
	}
}
