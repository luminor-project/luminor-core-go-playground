package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMagicLinkTokenNotFound     = errors.New("magic link token not found")
	ErrMagicLinkTokenExpired      = errors.New("magic link token expired")
	ErrMagicLinkTokenAlreadyUsed  = errors.New("magic link token already used")
	ErrMagicLinkRateLimitExceeded = errors.New("magic link rate limit exceeded")
)

const (
	MagicLinkTokenEntropyBytes = 32 // 256 bits
	MagicLinkTokenLifetime     = 15 * time.Minute
	MaxMagicLinksPerHour       = 3
)

// MagicLinkToken represents a time-limited, single-use authentication token.
type MagicLinkToken struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsValid checks if the token is unexpired and unused.
func (t MagicLinkToken) IsValid(now time.Time) bool {
	if t.UsedAt != nil {
		return false
	}
	return now.Before(t.ExpiresAt)
}

// HashToken creates a SHA-256 hash of a raw token for storage.
func HashToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}

// GenerateMagicLinkToken creates a new cryptographically secure magic link token.
func GenerateMagicLinkToken(accountID string, clock Clock) (MagicLinkToken, string, error) {
	// Generate cryptographically secure random bytes
	rawBytes := make([]byte, MagicLinkTokenEntropyBytes)
	if _, err := rand.Read(rawBytes); err != nil {
		return MagicLinkToken{}, "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Encode to URL-safe base64
	rawToken := base64.URLEncoding.EncodeToString(rawBytes)

	token := MagicLinkToken{
		ID:        uuid.New().String(),
		AccountID: accountID,
		TokenHash: HashToken(rawToken),
		ExpiresAt: clock.Now().Add(MagicLinkTokenLifetime),
		CreatedAt: clock.Now(),
	}

	return token, rawToken, nil
}
