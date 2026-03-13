package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetToken represents a time-limited token for password recovery.
type PasswordResetToken struct {
	ID        string
	AccountID string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// NewPasswordResetToken creates a token with 24h expiration.
// Returns the entity and the raw token (for email only).
func NewPasswordResetToken(accountID string, clock Clock) (PasswordResetToken, string, error) {
	rawToken := uuid.New().String() + uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.DefaultCost)
	if err != nil {
		return PasswordResetToken{}, "", err
	}

	return PasswordResetToken{
		ID:        uuid.New().String(),
		AccountID: accountID,
		TokenHash: string(hash),
		ExpiresAt: clock.Now().Add(24 * time.Hour),
		CreatedAt: clock.Now(),
	}, rawToken, nil
}

// IsValid checks if token is unused and not expired.
func (t PasswordResetToken) IsValid(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}

// Verify checks if raw token matches the hash.
func (t PasswordResetToken) Verify(rawToken string) bool {
	return bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(rawToken)) == nil
}

// MarkUsed records consumption time.
func (t *PasswordResetToken) MarkUsed(now time.Time) {
	t.UsedAt = &now
}
