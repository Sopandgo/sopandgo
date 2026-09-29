package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrInvalidKey indicates the encryption key material is missing or wrong length.
var ErrInvalidKey = errors.New("encryption key must be 32 bytes (AES-256)")

// Seal encrypts plaintext with AES-GCM. Output format: nonce || ciphertext.
func Seal(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(sealed))
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

// Open decrypts output from Seal.
func Open(key, blob []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

// KeyFromEnv reads SMTP_SECRET_ENCRYPTION_KEY: base64 or hex-encoded 32 bytes.
// Returns (nil, false) when unset.
func KeyFromEnv() ([]byte, bool) {
	s := strings.TrimSpace(os.Getenv("SMTP_SECRET_ENCRYPTION_KEY"))
	if s == "" {
		return nil, false
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == 32 {
		return b, true
	}
	if b, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err == nil && len(b) == 32 {
		return b, true
	}
	// Allow raw 32-byte UTF-8 string (discouraged but convenient for local dev)
	if len([]byte(s)) == 32 {
		b := []byte(s)
		return b, true
	}
	return nil, false
}
