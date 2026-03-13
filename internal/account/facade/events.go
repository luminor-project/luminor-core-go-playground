package facade

import "time"

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}

// PasswordResetRequestedEvent signals that a reset email should be sent.
// Note: The raw token is NOT included in this event - only the pre-built ResetURL contains it.
type PasswordResetRequestedEvent struct {
	AccountID string
	Email     string
	ResetURL  string    // Pre-built URL with encoded token (for email only)
	TokenID   string    // For correlation/tracking (safe to log)
	ExpiresAt time.Time // When the reset link expires
}

// PasswordResetCompletedEvent signals successful password change.
type PasswordResetCompletedEvent struct {
	AccountID string
	Email     string
}
