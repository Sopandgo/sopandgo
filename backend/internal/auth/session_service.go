package auth

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/google/uuid"
)

// StartSessionsCleanupTask launches a background cleanup loop.
// It accepts a Context so the main function can stop it gracefully.
func (s *Service) StartSessionsCleanupTask(ctx context.Context, interval time.Duration) {
	go func() {
		// 1. Run immediately on startup (don't wait for the first tick)
		if err := s.cleanup(); err != nil {
			log.Printf("Initial session cleanup failed: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 2. Run on interval
				if err := s.cleanup(); err != nil {
					log.Printf("Session cleanup error: %v", err)
				}

			case <-ctx.Done():
				// 3. Stop gracefully when the app shuts down
				log.Println("Stopping session cleanup task...")
				return
			}
		}
	}()
}

// Helper to keep the goroutine logic clean
func (s *Service) cleanup() error {
	return cleanupExpiredTokensRecord(s.db)
}

func (s *Service) CreateSession(userID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenID := uuid.NewString()
	createdAt := time.Now().UTC()
	expiresAt := createdAt.Add(7 * 24 * time.Hour) // 7-day session

	if err := insertRefreshTokenRecord(s.db, tokenID, userID, expiresAt, createdAt); err != nil {
		return "", err
	}

	return tokenID, nil
}

func (s *Service) ListActiveSessions() ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return listActiveSessionsRecord(s.db)
}

// ValidateSession checks if a refresh token is valid and returns the user's credentials
// needed to mint a new access token.
func (s *Service) ValidateSession(tokenID string) (userID, role string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return validateRefreshTokenRecord(s.db, tokenID)
}

// RevokeSession revokes a single session (Logout)
func (s *Service) RevokeSession(tokenString string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. LOOKUP
	ownerID, err := getActiveTokenOwnerRecord(s.db, tokenString)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	// Start Transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 2. REVOKE
	if err := revokeRefreshTokenRecord(tx, tokenString); err != nil {
		return err
	}

	// 3. DETERMINE ACTOR
	finalActor := actorUserID
	if finalActor == nil {
		finalActor = &ownerID
	}

	// 4. PREPARE SAFE LOG DATA
	// SECURITY FIX: Never log the full token!
	safeTokenMask := "REDACTED"
	if len(tokenString) > 6 {
		// Keep only last 4 chars for debugging (e.g. "...a9b1")
		safeTokenMask = "..." + tokenString[len(tokenString)-4:]
	}

	// 5. AUDIT
	err = s.auditLogger.Log(
		tx,
		audit.EventLogout,
		audit.EntityUser,
		ownerID,
		finalActor,
		map[string]string{
			"token_mask": safeTokenMask, // Log the mask, not the secret
			"status":     "success",
		},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RevokeAllUserSessions revokes all sessions for a specific user
func (s *Service) RevokeAllUserSessions(targetUserID string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Revoke
	if err := revokeUserRefreshTokensRecord(tx, targetUserID); err != nil {
		return err
	}

	// 2. Audit
	err = s.auditLogger.Log(
		tx,
		audit.EventAdminRevokedAllUserSessions,
		audit.EntityUser,
		targetUserID,
		actorUserID,
		map[string]string{"status": "revoked_all"},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RevokeAllSessions is the global kill-switch
// reason strictly required
func (s *Service) RevokeAllSessions(reason string, actorUserID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Revoke Global
	if err := revokeAllRefreshTokensRecord(tx); err != nil {
		return err
	}

	// 2. Audit
	err = s.auditLogger.Log(
		tx,
		audit.EventSystemGlobalRevocation,
		audit.EntitySystem,
		"all_sessions",
		actorUserID,
		map[string]string{"reason": reason},
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
