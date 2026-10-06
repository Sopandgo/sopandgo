package i18n

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"strings"
	"sync"
)

const Base = "en"

// ErrUnsupportedLocale is returned when a tag is not in the shipped list.
var ErrUnsupportedLocale = errors.New("unsupported locale")

//go:embed supported-locales.json
var supportedLocalesJSON []byte

//go:embed messages/*.json
var messageFiles embed.FS

var (
	loadOnce sync.Once
	locales  []string
	allowed  map[string]struct{}
	bundles  map[string]map[string]string
)

func load() {
	loadOnce.Do(func() {
		if err := json.Unmarshal(supportedLocalesJSON, &locales); err != nil {
			panic("i18n: supported-locales.json: " + err.Error())
		}
		allowed = make(map[string]struct{}, len(locales))
		bundles = make(map[string]map[string]string, len(locales))
		for _, tag := range locales {
			allowed[tag] = struct{}{}
			raw, err := messageFiles.ReadFile("messages/" + tag + ".json")
			if err != nil {
				panic("i18n: missing messages/" + tag + ".json")
			}
			var msgs map[string]string
			if err := json.Unmarshal(raw, &msgs); err != nil {
				panic("i18n: messages/" + tag + ".json: " + err.Error())
			}
			bundles[tag] = msgs
		}
		if _, ok := allowed[Base]; !ok {
			panic("i18n: base locale " + Base + " is not in supported-locales.json")
		}
	})
}

// Supported returns the shipped locale tags in list order.
func Supported() []string {
	load()
	out := make([]string, len(locales))
	copy(out, locales)
	return out
}

// Normalize reports whether tag is a supported locale. Matching is exact after trim and lowercasing.
func Normalize(tag string) (string, bool) {
	load()
	tag = strings.ToLower(strings.TrimSpace(tag))
	if _, ok := allowed[tag]; !ok {
		return "", false
	}
	return tag, true
}

// Fallback returns tag when it is supported, otherwise the base locale.
func Fallback(tag string) string {
	if n, ok := Normalize(tag); ok {
		return n
	}
	return Base
}

// T renders a message. Missing keys and unknown locales fall back to English.
// Placeholders use {name} so word order can change per language.
func T(locale, key string, args map[string]string) string {
	load()
	locale = Fallback(locale)
	msg, ok := bundles[locale][key]
	if !ok {
		msg = bundles[Base][key]
	}
	if msg == "" && !ok {
		return key
	}
	for name, value := range args {
		msg = strings.ReplaceAll(msg, "{"+name+"}", value)
	}
	return msg
}

// ReadDefaultLocale reads app_settings.default_locale, falling back to English.
func ReadDefaultLocale(db *sql.DB) string {
	if db == nil {
		return Base
	}
	var tag string
	err := db.QueryRow(`SELECT default_locale FROM app_settings WHERE id = 1`).Scan(&tag)
	if err != nil {
		return Base
	}
	return Fallback(tag)
}
