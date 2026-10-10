package auth

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
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

func decodeAvatarImage(raw []byte) (image.Image, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty image")
	}
	if !sniffAllowedAvatar(raw) {
		return nil, fmt.Errorf("unsupported image type")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("invalid image dimensions")
	}
	if w > maxAvatarPixelEdge || h > maxAvatarPixelEdge {
		return nil, fmt.Errorf("image too large")
	}
	return img, nil
}

func sniffAllowedAvatar(raw []byte) bool {
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
	img, err := decodeAvatarImage(raw)
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
