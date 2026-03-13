package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMagicLinkNotFound = errors.New("magic link not found")
	ErrMagicLinkExpired  = errors.New("magic link expired")
	ErrMagicLinkUsed     = errors.New("magic link already used")
	ErrTooManyMagicLinks = errors.New("too many active magic links")
)

// MagicLinkToken represents a single-use, time-limited authentication token.
type MagicLinkToken struct {
	ID        string
	AccountID string
	TokenHash string // SHA-256 hash of the plaintext token
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// NewMagicLinkToken creates a new magic link token with the given account ID and expiration time.
// Returns the token entity and the plaintext token (which should be sent to the user).
func NewMagicLinkToken(accountID string, expiresAt time.Time, now time.Time) (MagicLinkToken, string, error) {
	plaintextToken, err := generateSecureToken(32)
	if err != nil {
		return MagicLinkToken{}, "", fmt.Errorf("generate token: %w", err)
	}

	tokenHash := hashToken(plaintextToken)

	token := MagicLinkToken{
		ID:        uuid.New().String(),
		AccountID: accountID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	return token, plaintextToken, nil
}

// IsExpired returns true if the token has expired based on the provided time.
func (t MagicLinkToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t MagicLinkToken) IsUsed() bool {
	return t.UsedAt != nil
}

// MarkUsed marks the token as consumed at the provided time.
func (t *MagicLinkToken) MarkUsed(now time.Time) {
	t.UsedAt = &now
}

// ValidateToken validates a plaintext token against the stored hash using constant-time comparison.
func (t MagicLinkToken) ValidateToken(plaintextToken string) bool {
	hash := hashToken(plaintextToken)
	return subtle.ConstantTimeCompare([]byte(t.TokenHash), []byte(hash)) == 1
}

// generateSecureToken generates a cryptographically secure random token of the specified byte length.
// Returns a URL-safe base64-encoded string.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of the token for secure storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}
