package auth

import (
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
)

// symmetricKey is generated automatically every time the server starts.
// This means all existing tokens become invalid after a restart.
var symmetricKey = paseto.NewV4SymmetricKey()

func GenerateAccessToken(userID, role string) (string, error) {
	token := paseto.NewToken()

	// Set standard claims
	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	token.SetExpiration(time.Now().Add(5 * time.Minute))

	// Set custom claims
	token.SetString("user_id", userID)
	token.SetString("role", role)

	// Encrypt
	return token.V4Encrypt(symmetricKey, nil), nil
}

func VerifyToken(tokenStr string) (*TokenClaims, error) {
	// Create a parser to validate standard claims
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())
	parser.AddRule(paseto.ValidAt(time.Now()))

	// Decrypt and validate
	token, err := parser.ParseV4Local(symmetricKey, tokenStr, nil)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Extract custom claims
	userID, err := token.GetString("user_id")
	if err != nil {
		return nil, fmt.Errorf("user_id missing: %w", err)
	}
	role, err := token.GetString("role")
	if err != nil {
		return nil, fmt.Errorf("role missing: %w", err)
	}

	return &TokenClaims{
		UserID:   userID,
		UserRole: role,
	}, nil
}
