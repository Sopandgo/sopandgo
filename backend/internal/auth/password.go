package auth

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	BcryptCost = 12
	MinPassLen = 12
	MaxPassLen = 128
)

// Common validation errors
var (
	ErrPasswordTooShort   = fmt.Errorf("password must be at least %d characters", MinPassLen)
	ErrPasswordTooLong    = fmt.Errorf("password must be at most %d characters", MaxPassLen)
	ErrPasswordWhitespace = errors.New("password must not start or end with whitespace")
	ErrPasswordTooSimple  = errors.New("password must include at least 3 of: lowercase, uppercase, number, symbol")
	ErrPasswordRepetitive = errors.New("password is too repetitive")
	ErrPasswordEmpty      = errors.New("password cannot be empty")
)

func HashPassword(plainText string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plainText), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func VerifyPassword(plainText, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainText))
}

func ValidatePassword(pw string) error {
	if pw == "" {
		return ErrPasswordEmpty
	}

	if len(pw) < MinPassLen {
		return ErrPasswordTooShort
	}
	if len(pw) > MaxPassLen {
		return ErrPasswordTooLong
	}

	// Disallow leading/trailing whitespace
	if strings.TrimSpace(pw) != pw {
		return ErrPasswordWhitespace
	}

	// Character class variety: require at least 3 of 4
	var (
		hasLower  bool
		hasUpper  bool
		hasDigit  bool
		hasSymbol bool
	)

	for _, r := range pw {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		default:
			hasSymbol = true
		}
	}

	classes := 0
	if hasLower {
		classes++
	}
	if hasUpper {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSymbol {
		classes++
	}

	if classes < 3 {
		return ErrPasswordTooSimple
	}

	return nil
}
