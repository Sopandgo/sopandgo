// Package imageutil decodes uploaded raster images (JPEG, PNG, WebP) with the
// checks every upload path needs: a magic-byte sniff, then a size check on the
// header before the pixels are decoded.
package imageutil

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/sopandgo/sopandgo/backend/internal/errkind"
	_ "golang.org/x/image/webp"
)

var (
	// ErrInvalidImage refuses data that is empty, not a JPEG, PNG or WebP
	// file, can't be decoded, or has no pixels.
	ErrInvalidImage = errkind.New(errkind.Invalid, "invalid image")
	// ErrImageTooLarge refuses an image with a side longer than the limit.
	ErrImageTooLarge = errkind.New(errkind.Invalid, "image too large")
)

// Sniff reports whether raw starts like a JPEG, PNG, or WebP file.
func Sniff(raw []byte) bool {
	n := 12
	if len(raw) < n {
		n = len(raw)
	}
	head := raw[:n]
	// JPEG
	if len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff {
		return true
	}
	// PNG
	if len(head) >= 8 && bytes.Equal(head[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		return true
	}
	// WebP: RIFF....WEBP
	if len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return true
	}
	return false
}

// Decode checks the type and dimensions, then decodes the image. Images with a
// side longer than maxEdge are refused before their pixels are allocated.
// It refuses with ErrImageTooLarge or ErrInvalidImage.
func Decode(raw []byte, maxEdge int) (image.Image, error) {
	if len(raw) == 0 {
		return nil, errkind.Detail(ErrInvalidImage, "empty image")
	}
	if !Sniff(raw) {
		return nil, errkind.Detail(ErrInvalidImage, "unsupported image type")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, errkind.Detail(ErrInvalidImage, "invalid image: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return nil, errkind.Detail(ErrInvalidImage, "invalid image dimensions")
	}
	if cfg.Width > maxEdge || cfg.Height > maxEdge {
		return nil, errkind.Detail(ErrImageTooLarge, "image too large")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, errkind.Detail(ErrInvalidImage, "invalid image: %w", err)
	}
	return img, nil
}
