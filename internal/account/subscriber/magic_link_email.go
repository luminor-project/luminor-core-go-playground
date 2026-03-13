package subscriber

import (
	"context"
	"fmt"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	SendMagicLink(ctx context.Context, email, rawToken string, expiresAt interface{}) error
}

// MagicLinkEmailHandler handles magic link requested events and sends emails.
type MagicLinkEmailHandler struct {
	sender EmailSender
}

// NewMagicLinkEmailHandler creates a new magic link email handler.
func NewMagicLinkEmailHandler(sender EmailSender) *MagicLinkEmailHandler {
	return &MagicLinkEmailHandler{sender: sender}
}

// RegisterMagicLinkSubscriber subscribes to MagicLinkRequestedEvent and sends emails.
func RegisterMagicLinkSubscriber(bus *eventbus.Bus, handler *MagicLinkEmailHandler) {
	eventbus.Subscribe(bus, func(ctx context.Context, e facade.MagicLinkRequestedEvent) error {
		if err := handler.sender.SendMagicLink(ctx, e.Email, e.RawToken, e.ExpiresAt); err != nil {
			return fmt.Errorf("send magic link email to %s: %w", e.Email, err)
		}
		return nil
	})
}
