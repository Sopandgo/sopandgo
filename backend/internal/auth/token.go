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

// challengeKey signs the short-lived token that sits between the password and
// the second factor. It is separate from symmetricKey, so a challenge token
// can never be accepted as an access token, and it also resets on restart.
var challengeKey = paseto.NewV4SymmetricKey()

const (
	challengePurpose = "two_factor_challenge"
	// TwoFactorChallengeTTL is how long a user has to enter the second factor.
	TwoFactorChallengeTTL = 5 * time.Minute
)

// GenerateTwoFactorChallenge is issued after a correct password when the
// account has 2FA on. It proves only the password step.
func GenerateTwoFactorChallenge(userID string) (string, error) {
	token := paseto.NewToken()
	now := time.Now()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(TwoFactorChallengeTTL))
	token.SetString("user_id", userID)
	token.SetString("purpose", challengePurpose)
	return token.V4Encrypt(challengeKey, nil), nil
}

// VerifyTwoFactorChallenge returns the user ID from a valid challenge token.
func VerifyTwoFactorChallenge(tokenStr string) (string, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())
	parser.AddRule(paseto.ValidAt(time.Now()))
	token, err := parser.ParseV4Local(challengeKey, tokenStr, nil)
	if err != nil {
		return "", fmt.Errorf("challenge validation failed: %w", err)
	}
	purpose, err := token.GetString("purpose")
	if err != nil || purpose != challengePurpose {
		return "", fmt.Errorf("not a two-factor challenge")
	}
	userID, err := token.GetString("user_id")
	if err != nil {
		return "", fmt.Errorf("user_id missing: %w", err)
	}
	return userID, nil
}
