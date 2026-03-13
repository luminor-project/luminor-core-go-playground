package domain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"crypto/sha256"

	"github.com/google/uuid"
)

// PasswordResetToken represents a time-limited token for password recovery.
type PasswordResetToken struct {
	ID        string
	AccountID string
	TokenHash string // SHA-256 hash (64 hex characters)
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// generateSecureToken creates a cryptographically secure random token.
// Returns 32 bytes encoded as URL-safe base64 (43 characters).
func generateSecureToken() (string, error) {
	// 32 bytes = 256 bits of entropy
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	// Encode to URL-safe base64 (no padding)
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// hashToken computes SHA-256 hash of a token for storage.
func hashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// NewPasswordResetToken creates a token with 24h expiration.
// Returns the entity and the raw token (for email only).
func NewPasswordResetToken(accountID string, clock Clock) (PasswordResetToken, string, error) {
	rawToken, err := generateSecureToken()
	if err != nil {
		return PasswordResetToken{}, "", err
	}

	return PasswordResetToken{
		ID:        uuid.New().String(),
		AccountID: accountID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: clock.Now().Add(24 * time.Hour),
		CreatedAt: clock.Now(),
	}, rawToken, nil
}

// IsValid checks if token is unused and not expired.
func (t PasswordResetToken) IsValid(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}

// Verify performs constant-time comparison of raw token against stored hash.
// This prevents timing attacks by always taking the same amount of time.
func (t PasswordResetToken) Verify(rawToken string) bool {
	computedHash := hashToken(rawToken)
	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(t.TokenHash), []byte(computedHash)) == 1
}

// MarkUsed records consumption time.
func (t *PasswordResetToken) MarkUsed(now time.Time) {
	t.UsedAt = &now
}
