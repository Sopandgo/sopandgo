package errkind_test

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
)

func TestNewMatchesItsKindOnly(t *testing.T) {
	err := errkind.New(errkind.Conflict, "this SOP is already archived")

	if !errors.Is(err, errkind.Conflict) {
		t.Fatalf("want a conflict, got %v", err)
	}
	if errors.Is(err, errkind.NotFound) || errors.Is(err, errkind.Invalid) {
		t.Fatalf("a conflict must not match other kinds")
	}
	if got := err.Error(); got != "this SOP is already archived" {
		t.Fatalf("text = %q, want the message without a prefix", got)
	}
}

func TestNewFormatsAndWrapsACause(t *testing.T) {
	err := errkind.New(errkind.Invalid, "invalid image: %w", fs.ErrInvalid)

	if !errors.Is(err, errkind.Invalid) {
		t.Fatalf("want invalid, got %v", err)
	}
	if !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("the cause must stay reachable")
	}
	if got := err.Error(); got != "invalid image: invalid argument" {
		t.Fatalf("text = %q", got)
	}
}

func TestKindSurvivesCallerWrapping(t *testing.T) {
	named := errkind.New(errkind.NotFound, "no avatar")
	wrapped := fmt.Errorf("remove avatar: %w", named)

	if !errors.Is(wrapped, errkind.NotFound) || !errors.Is(wrapped, named) {
		t.Fatalf("want both the kind and the named error through %%w, got %v", wrapped)
	}
	kind, ok := errkind.KindOf(wrapped)
	if !ok || kind != errkind.NotFound {
		t.Fatalf("KindOf = %v, %v", kind, ok)
	}
}

func TestKindOfPlainErrorHasNoKind(t *testing.T) {
	if _, ok := errkind.KindOf(errors.New("sop not found: but not marked")); ok {
		t.Fatalf("an unmarked error must have no kind, whatever its text")
	}
	if _, ok := errkind.KindOf(nil); ok {
		t.Fatalf("nil has no kind")
	}
}

func TestKindTextNamesTheKind(t *testing.T) {
	for kind, want := range map[errkind.Kind]string{
		errkind.Conflict: "conflict",
		errkind.NotFound: "not found",
		errkind.Invalid:  "invalid",
	} {
		if kind.Error() != want {
			t.Errorf("%v.Error() = %q, want %q", kind, kind.Error(), want)
		}
	}
}

func TestDetailMatchesTheNamedErrorAndItsKind(t *testing.T) {
	ErrAssetReference := errkind.New(errkind.Invalid, "invalid asset reference")
	err := errkind.Detail(ErrAssetReference, "referenced asset not found: %s", "assets/x.png")

	if got := err.Error(); got != "referenced asset not found: assets/x.png" {
		t.Fatalf("text = %q, want the detail message alone", got)
	}
	if !errors.Is(err, ErrAssetReference) {
		t.Fatalf("want the named error")
	}
	if kind, ok := errkind.KindOf(fmt.Errorf("create version: %w", err)); !ok || kind != errkind.Invalid {
		t.Fatalf("want the named error's kind through wrapping, got %v %v", kind, ok)
	}
	if errors.Is(err, errkind.New(errkind.Invalid, "invalid asset reference")) {
		t.Fatalf("a different named error with the same text must not match")
	}
}

func TestDetailWrapsACause(t *testing.T) {
	ErrInvalidImage := errkind.New(errkind.Invalid, "invalid image")
	err := errkind.Detail(ErrInvalidImage, "invalid image: %w", fs.ErrInvalid)

	if !errors.Is(err, fs.ErrInvalid) || !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("want both the cause and the named error, got %v", err)
	}
}
