package imageutil_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	"github.com/sopandgo/sopandgo/backend/internal/imageutil"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestDecodeRefusesUnusableImagesAsInvalid(t *testing.T) {
	good := encodePNG(t, 4, 4)
	for name, raw := range map[string][]byte{
		"empty":       nil,
		"unsupported": []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"/>"),
		"undecodable": good[:20],
	} {
		_, err := imageutil.Decode(raw, 100)
		if !errors.Is(err, imageutil.ErrInvalidImage) || !errors.Is(err, errkind.Invalid) {
			t.Errorf("%s: want invalid image, got %v", name, err)
		}
		if errors.Is(err, imageutil.ErrImageTooLarge) {
			t.Errorf("%s: must not be image too large", name)
		}
	}
}

func TestDecodeRefusesOversizedImagesAsTooLarge(t *testing.T) {
	_, err := imageutil.Decode(encodePNG(t, 101, 1), 100)
	if !errors.Is(err, imageutil.ErrImageTooLarge) || !errors.Is(err, errkind.Invalid) {
		t.Fatalf("want image too large, got %v", err)
	}
	if errors.Is(err, imageutil.ErrInvalidImage) {
		t.Fatalf("an oversized image must not be invalid image")
	}
}

func TestDecodeAcceptsAnImageWithinTheLimit(t *testing.T) {
	img, err := imageutil.Decode(encodePNG(t, 100, 3), 100)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 100 || b.Dy() != 3 {
		t.Fatalf("bounds = %v", b)
	}
}
