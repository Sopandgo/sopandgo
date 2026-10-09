package secrets

import (
	"bytes"
	"strings"
	"testing"
)

func TestSealOpen_RoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	plain := []byte("smtp-secret-password")
	enc, err := Seal(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Open(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("got %q want %q", out, plain)
	}
}

func TestOpen_WrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	wrong := bytes.Repeat([]byte{2}, 32)
	enc, err := Seal(key, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Open(wrong, enc)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestKeyFromEnv_NameAndLegacyFallback(t *testing.T) {
	newKey := strings.Repeat("a", 32)
	oldKey := strings.Repeat("b", 32)

	cases := []struct {
		name, current, legacy string
		wantSource            string
		wantKey               string
	}{
		{"unset", "", "", "", ""},
		{"new name", newKey, "", KeyEnv, newKey},
		{"legacy name only", "", oldKey, LegacyKeyEnv, oldKey},
		{"new name wins", newKey, oldKey, KeyEnv, newKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(KeyEnv, tc.current)
			t.Setenv(LegacyKeyEnv, tc.legacy)
			if got := KeyEnvSource(); got != tc.wantSource {
				t.Fatalf("KeyEnvSource() = %q, want %q", got, tc.wantSource)
			}
			key, ok := KeyFromEnv()
			if ok != (tc.wantKey != "") || string(key) != tc.wantKey {
				t.Fatalf("KeyFromEnv() = %q, %v; want %q", key, ok, tc.wantKey)
			}
		})
	}
}
