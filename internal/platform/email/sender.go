// Package email provides email sending capabilities.
// For the initial implementation, emails are logged to stdout.
// In production, this should be replaced with an actual email provider (SendGrid, AWS SES, etc.).
package email

import (
	"context"
	"fmt"
	"log/slog"
)

// Sender is the interface for sending emails.
type Sender interface {
	Send(ctx context.Context, to, subject, bodyText, bodyHTML string) error
}

// LogSender logs emails to stdout instead of actually sending them.
// This is useful for development and testing.
type LogSender struct{}

// NewLogSender creates a new LogSender.
func NewLogSender() *LogSender {
	return &LogSender{}
}

// Send logs the email to stdout.
func (s *LogSender) Send(ctx context.Context, to, subject, bodyText, bodyHTML string) error {
	slog.Info("EMAIL",
		"to", to,
		"subject", subject,
		"body_text", bodyText,
		"body_html", bodyHTML,
	)
	return nil
}

// MagicLinkEmailData holds the data for a magic link email.
type MagicLinkEmailData struct {
	MagicLinkURL string
	ExpiresAt    string
}

// RenderMagicLinkEmail renders the magic link email templates.
func RenderMagicLinkEmail(data MagicLinkEmailData) (subject, bodyText, bodyHTML string) {
	subject = "Your magic login link"

	bodyText = fmt.Sprintf(`Hi there,

Click the link below to log in to your account:

%s

This link will expire at %s and can only be used once.

If you didn't request this link, you can safely ignore this email.
`, data.MagicLinkURL, data.ExpiresAt)

	bodyHTML = fmt.Sprintf(`<p>Hi there,</p>

<p>Click the link below to log in to your account:</p>

<p><a href="%s">Log In Now</a></p>

<p>This link will expire at %s and can only be used once.</p>

<p>If you didn't request this link, you can safely ignore this email.</p>
`, data.MagicLinkURL, data.ExpiresAt)

	return subject, bodyText, bodyHTML
}
