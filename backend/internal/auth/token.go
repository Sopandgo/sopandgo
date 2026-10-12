package auth

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
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

// ErrChallengeUsed is returned for a challenge that already opened a session.
var ErrChallengeUsed = errors.New("two-factor challenge already used")

// ChallengeClaims identifies one password step: the user and a random ID.
type ChallengeClaims struct {
	UserID    string
	ID        string
	ExpiresAt time.Time
}

// usedChallenges holds the IDs of challenges that opened a session, until they
// expire, so one password step opens at most one session. It lives in memory
// like challengeKey: a restart invalidates every challenge anyway.
var usedChallenges = struct {
	sync.Mutex
	ids map[string]time.Time // challenge ID -> expiry
}{ids: map[string]time.Time{}}

// GenerateTwoFactorChallenge is issued after a correct password when a second
// factor is required. It proves only the password step.
func GenerateTwoFactorChallenge(userID string) (string, error) {
	token := paseto.NewToken()
	now := time.Now()
	token.SetIssuedAt(now)
	token.SetNotBefore(now)
	token.SetExpiration(now.Add(TwoFactorChallengeTTL))
	token.SetJti(uuid.NewString())
	token.SetString("user_id", userID)
	token.SetString("purpose", challengePurpose)
	return token.V4Encrypt(challengeKey, nil), nil
}

// VerifyTwoFactorChallenge checks a challenge token and returns its claims.
// A challenge that already opened a session returns ErrChallengeUsed.
func VerifyTwoFactorChallenge(tokenStr string) (*ChallengeClaims, error) {
	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())
	parser.AddRule(paseto.ValidAt(time.Now()))
	token, err := parser.ParseV4Local(challengeKey, tokenStr, nil)
	if err != nil {
		return nil, fmt.Errorf("challenge validation failed: %w", err)
	}
	purpose, err := token.GetString("purpose")
	if err != nil || purpose != challengePurpose {
		return nil, fmt.Errorf("not a two-factor challenge")
	}
	userID, err := token.GetString("user_id")
	if err != nil {
		return nil, fmt.Errorf("user_id missing: %w", err)
	}
	id, err := token.GetJti()
	if err != nil || id == "" {
		return nil, fmt.Errorf("challenge id missing")
	}
	exp, err := token.GetExpiration()
	if err != nil {
		return nil, fmt.Errorf("expiration missing: %w", err)
	}

	usedChallenges.Lock()
	_, used := usedChallenges.ids[id]
	usedChallenges.Unlock()
	if used {
		return nil, ErrChallengeUsed
	}
	return &ChallengeClaims{UserID: userID, ID: id, ExpiresAt: exp}, nil
}

// ConsumeTwoFactorChallenge marks a challenge as used. It returns false when
// it was already used, so of two requests racing with different valid codes,
// only one opens a session. Call it after the second factor succeeded, so a
// typo does not cost the user their password step.
func ConsumeTwoFactorChallenge(c *ChallengeClaims) bool {
	usedChallenges.Lock()
	defer usedChallenges.Unlock()

	now := time.Now()
	for id, exp := range usedChallenges.ids {
		if now.After(exp) {
			delete(usedChallenges.ids, id)
		}
	}
	if _, used := usedChallenges.ids[c.ID]; used {
		return false
	}
	usedChallenges.ids[c.ID] = c.ExpiresAt
	return true
}
