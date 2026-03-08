package facade

// AccountCreatedEvent is dispatched when a new account is registered.
type AccountCreatedEvent struct {
	AccountID string
	Email     string
}
