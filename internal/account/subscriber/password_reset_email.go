package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/outbox"
)

// outboxStore is the interface for enqueuing outbox events.
type outboxStore interface {
	Enqueue(ctx context.Context, eventType string, payload any) error
}

// RegisterPasswordResetEmailSubscriber wires the email handler for password reset requests.
func RegisterPasswordResetEmailSubscriber(bus *eventbus.Bus, store outboxStore, baseURL string) {
	eventbus.Subscribe(bus, func(ctx context.Context, e facade.PasswordResetRequestedEvent) error {
		slog.Info("handling PasswordResetRequestedEvent",
			"account_id", e.AccountID,
			"email", e.Email,
		)

		// Build reset URL: https://example.com/reset-password?token=<rawToken>
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, e.RawToken)

		// Enqueue email via outbox for async delivery
		email := passwordResetEmail{
			To:       e.Email,
			ResetURL: resetURL,
		}

		if err := store.Enqueue(ctx, outbox.EventTypeSendEmailV1, email); err != nil {
			slog.Error("failed to enqueue password reset email", "error", err, "email", e.Email)
			return fmt.Errorf("enqueue password reset email: %w", err)
		}

		return nil
	})
}

type passwordResetEmail struct {
	To       string `json:"to"`
	ResetURL string `json:"reset_url"`
}
