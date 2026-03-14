// Package email provides email sending capabilities.
package email

import "context"

// Sender is the interface for sending emails.
type Sender interface {
	// SendPasswordReset sends a password reset email with the given reset URL.
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

// NoOpSender is a sender that does nothing (for testing/development).
type NoOpSender struct{}

func (n *NoOpSender) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	return nil
}
