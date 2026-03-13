package facade

import "time"

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}

// MagicLinkRequestedEvent is dispatched when a user requests a magic link.
type MagicLinkRequestedEvent struct {
	AccountID    string
	Email        string
	MagicLinkURL string
	ExpiresAt    time.Time
}
