// Package subscriber provides event subscribers for sending emails.
package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/email"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

// MagicLinkSubscriber handles MagicLinkRequestedEvent and sends emails.
type MagicLinkSubscriber struct {
	sender email.Sender
}

// NewMagicLinkSubscriber creates a new MagicLinkSubscriber.
func NewMagicLinkSubscriber(sender email.Sender) *MagicLinkSubscriber {
	return &MagicLinkSubscriber{sender: sender}
}

// Register registers the subscriber with the event bus.
func (s *MagicLinkSubscriber) Register(bus *eventbus.Bus) {
	eventbus.Subscribe(bus, func(ctx context.Context, e accountfacade.MagicLinkRequestedEvent) error {
		return s.handleMagicLinkRequested(ctx, e)
	})
}

func (s *MagicLinkSubscriber) handleMagicLinkRequested(ctx context.Context, e accountfacade.MagicLinkRequestedEvent) error {
	slog.Info("handling MagicLinkRequestedEvent",
		"account_id", e.AccountID,
		"email", e.Email,
	)

	// Format expiration time for display
	expiresAtStr := e.ExpiresAt.Format(time.RFC1123)

	// Render email
	subject, bodyText, bodyHTML := email.RenderMagicLinkEmail(email.MagicLinkEmailData{
		MagicLinkURL: e.MagicLinkURL,
		ExpiresAt:    expiresAtStr,
	})

	// Send email
	if err := s.sender.Send(ctx, e.Email, subject, bodyText, bodyHTML); err != nil {
		return fmt.Errorf("send magic link email to %s: %w", e.Email, err)
	}

	slog.Info("magic link email sent successfully",
		"account_id", e.AccountID,
		"email", e.Email,
	)

	return nil
}
