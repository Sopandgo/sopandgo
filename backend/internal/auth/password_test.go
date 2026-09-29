package auth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerifyPassword(t *testing.T) {
	plainText := "SuperSecurePassword123!"

	// 1. Hash the password
	hash, err := auth.HashPassword(plainText)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	if hash == "" {
		t.Fatal("Expected a valid hash string, got empty")
	}
	if hash == plainText {
		t.Fatal("Hash matches plaintext (not hashed at all!)")
	}

	// 2. Verify with correct password
	err = auth.VerifyPassword(plainText, hash)
	if err != nil {
		t.Errorf("VerifyPassword failed for correct password: %v", err)
	}

	// 3. Verify with incorrect password
	err = auth.VerifyPassword("WrongPassword123!", hash)
	if err == nil {
		t.Error("VerifyPassword succeeded with incorrect password")
	}
	if !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		t.Errorf("Expected bcrypt mismatch error, got: %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	longPassword := strings.Repeat("A1b!", 35) // 140 characters

	tests := []struct {
		name        string
		password    string
		expectedErr error
	}{
		// Happy Paths (At least 12 chars, at least 3 classes)
		{"Valid: All 4 classes", "UpperLower123!", nil},
		{"Valid: Upper, Lower, Number", "UpperAndLower123", nil},
		{"Valid: Lower, Number, Symbol", "lowercase123456!", nil},
		{"Valid: Upper, Number, Symbol", "UPPERCASE123456!", nil},

		// Unhappy Paths: Length constraints
		{"Invalid: Empty", "", auth.ErrPasswordEmpty},
		{"Invalid: Too Short", "Ab1!", auth.ErrPasswordTooShort},
		{"Invalid: Too Long", longPassword, auth.ErrPasswordTooLong},

		// Unhappy Paths: Whitespace constraints
		{"Invalid: Leading space", " ValidPass123!", auth.ErrPasswordWhitespace},
		{"Invalid: Trailing space", "ValidPass123! ", auth.ErrPasswordWhitespace},

		// Unhappy Paths: Complexity constraints (less than 3 classes)
		{"Invalid: Only lowercase", "justlowercasewords", auth.ErrPasswordTooSimple},
		{"Invalid: Lower and numbers", "lowercase12345678", auth.ErrPasswordTooSimple},
		{"Invalid: Upper and numbers", "UPPERCASE12345678", auth.ErrPasswordTooSimple},
		{"Invalid: Lower and symbols", "lowercase!!!!!!!!", auth.ErrPasswordTooSimple},

		// Unhappy Paths: Repetitive
		{"Invalid: Fully Repetitive", "aaaaaaaaaaaa", auth.ErrPasswordTooSimple},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePassword(tt.password)
			if err != tt.expectedErr {
				t.Errorf("ValidatePassword(%q) error = %v; expected %v", tt.password, err, tt.expectedErr)
			}
		})
	}
}
