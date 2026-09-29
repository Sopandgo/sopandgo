package secrets

import (
	"bytes"
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
