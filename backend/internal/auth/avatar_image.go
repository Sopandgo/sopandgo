package auth

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"github.com/sopandgo/sopandgo/backend/internal/imageutil"
	"golang.org/x/image/draw"
)

const (
	maxAvatarUploadBytes = 5 << 20 // 5 MB
	maxAvatarPixelEdge   = 8192
	avatarJPEGQuality    = 85
)

var avatarPixelSizes = []int{96, 256, 512, 1024}

// AvatarSizeName maps API size query values to pixel edges.
var AvatarSizeName = map[string]int{
	"sm": 96,
	"md": 256,
	"lg": 512,
	"xl": 1024,
}

func avatarFileName(px int) string {
	return fmt.Sprintf("avatar-%d.jpg", px)
}

func centerCropSquare(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h < side {
		side = h
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2
	rect := image.Rect(0, 0, side, side)
	dst := image.NewRGBA(rect)
	draw.Draw(dst, rect, img, image.Pt(x0, y0), draw.Src)
	return dst
}

func resizeSquare(img image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: avatarJPEGQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func processAvatarSizes(raw []byte) (map[int][]byte, error) {
	img, err := imageutil.Decode(raw, maxAvatarPixelEdge)
	if err != nil {
		return nil, err
	}
	square := centerCropSquare(img)
	out := make(map[int][]byte, len(avatarPixelSizes))
	for _, px := range avatarPixelSizes {
		resized := resizeSquare(square, px)
		jpegBytes, err := encodeJPEG(resized)
		if err != nil {
			return nil, err
		}
		out[px] = jpegBytes
	}
	return out, nil
}
