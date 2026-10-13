// Package imageutil decodes uploaded raster images (JPEG, PNG, WebP) with the
// checks every upload path needs: a magic-byte sniff, then a size check on the
// header before the pixels are decoded.
package imageutil

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
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
//
// Error texts are matched by the HTTP handlers: "empty image",
// "unsupported image type", "invalid image", "invalid image dimensions",
// "image too large".
func Decode(raw []byte, maxEdge int) (image.Image, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty image")
	}
	if !Sniff(raw) {
		return nil, fmt.Errorf("unsupported image type")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return nil, fmt.Errorf("invalid image dimensions")
	}
	if cfg.Width > maxEdge || cfg.Height > maxEdge {
		return nil, fmt.Errorf("image too large")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}
	return img, nil
}
