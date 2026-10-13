package branding

import (
	"bytes"
	"image"
	"image/png"

	"github.com/sopandgo/sopandgo/backend/internal/imageutil"
	"golang.org/x/image/draw"
)

// processLogo decodes an upload and re-encodes it as two PNGs: the logo (aspect
// ratio kept, longest edge at most LogoMaxEdge, never upscaled) and a square
// IconSize icon with transparent padding. Re-encoding drops metadata and any
// bytes that are not pixels, so the public endpoint only ever serves plain PNG.
func processLogo(raw []byte) (logo, icon []byte, err error) {
	img, err := imageutil.Decode(raw, maxPixelEdge)
	if err != nil {
		return nil, nil, err
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	lw, lh := fit(w, h, LogoMaxEdge)
	logoImg := image.NewNRGBA(image.Rect(0, 0, lw, lh))
	draw.CatmullRom.Scale(logoImg, logoImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	iw, ih := fit(w, h, IconSize)
	iconImg := image.NewNRGBA(image.Rect(0, 0, IconSize, IconSize))
	x0, y0 := (IconSize-iw)/2, (IconSize-ih)/2
	draw.CatmullRom.Scale(iconImg, image.Rect(x0, y0, x0+iw, y0+ih), img, img.Bounds(), draw.Over, nil)

	if logo, err = encodePNG(logoImg); err != nil {
		return nil, nil, err
	}
	if icon, err = encodePNG(iconImg); err != nil {
		return nil, nil, err
	}
	return logo, icon, nil
}

// fit scales w×h so the longest side is at most edge, keeping at least 1 px.
func fit(w, h, edge int) (int, int) {
	if w <= edge && h <= edge {
		return w, h
	}
	if w >= h {
		return edge, max(1, h*edge/w)
	}
	return max(1, w*edge/h), edge
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
