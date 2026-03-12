package facade

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}

// PasswordResetRequestedEvent is dispatched when a password reset is requested.
type PasswordResetRequestedEvent struct {
	AccountID string
	Email     string
	ResetURL  string
}
