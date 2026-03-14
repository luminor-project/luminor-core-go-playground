package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrInvalidResetToken     = errors.New("invalid or expired reset token")
	ErrResetTokenExpired     = errors.New("reset token has expired")
	ErrResetTokenAlreadyUsed = errors.New("reset token has already been used")
)

// PasswordResetToken represents a password reset token entity.
type PasswordResetToken struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsValid checks if the token is valid (not expired and not used).
func (t PasswordResetToken) IsValid(now time.Time) bool {
	if t.UsedAt != nil {
		return false
	}
	return now.Before(t.ExpiresAt)
}

// MarkAsUsed marks the token as used at the given time.
func (t *PasswordResetToken) MarkAsUsed(now time.Time) {
	t.UsedAt = &now
}

// GenerateResetToken generates a cryptographically secure random token and its hash.
// Returns the plaintext token (to be sent to user) and the hash (to be stored in DB).
func GenerateResetToken() (plainToken, hashedToken string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", err
	}
	plainToken = base64.URLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(plainToken))
	hashedToken = hex.EncodeToString(hash[:])
	return plainToken, hashedToken, nil
}

// HashToken hashes a plaintext token for comparison.
func HashToken(plainToken string) string {
	hash := sha256.Sum256([]byte(plainToken))
	return hex.EncodeToString(hash[:])
}
