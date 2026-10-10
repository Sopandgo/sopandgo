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

func TestKeyFromEnv(t *testing.T) {
	raw := strings.Repeat("a", 32)

	cases := []struct {
		name, env string
		wantSet   bool
		wantKey   string
	}{
		{"unset", "", false, ""},
		{"valid", raw, true, raw},
		{"wrong length", "too-short", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(KeyEnv, tc.env)
			if got := KeySet(); got != tc.wantSet {
				t.Fatalf("KeySet() = %v, want %v", got, tc.wantSet)
			}
			key, ok := KeyFromEnv()
			if ok != (tc.wantKey != "") || string(key) != tc.wantKey {
				t.Fatalf("KeyFromEnv() = %q, %v; want %q", key, ok, tc.wantKey)
			}
		})
	}
}
