package facade

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}

// PasswordResetRequestedEvent signals that a reset email should be sent.
type PasswordResetRequestedEvent struct {
	AccountID string
	Email     string
	RawToken  string
}

// PasswordResetCompletedEvent signals successful password change.
type PasswordResetCompletedEvent struct {
	AccountID string
	Email     string
}
