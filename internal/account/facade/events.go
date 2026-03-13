package facade

import "time"

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}

// MagicLinkRequestedEvent is dispatched when a user requests a magic link.
type MagicLinkRequestedEvent struct {
	AccountID string
	Email     string
	RawToken  string
	ExpiresAt time.Time
}

// MagicLinkUsedEvent is dispatched when a magic link is successfully used.
type MagicLinkUsedEvent struct {
	AccountID string
	Email     string
	UsedAt    time.Time
}
