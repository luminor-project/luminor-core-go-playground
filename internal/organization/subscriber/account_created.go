package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type defaultOrgCreator interface {
	CreateDefaultOrg(ctx context.Context, accountID string) error
}

// RegisterAccountCreatedSubscriber subscribes to AccountCreatedEvent
// and creates a default organization for the new user.
func RegisterAccountCreatedSubscriber(bus *eventbus.Bus, orgCreator defaultOrgCreator) {
	eventbus.Subscribe(bus, func(ctx context.Context, e accountfacade.AccountCreatedEvent) error {
		slog.Info("handling AccountCreatedEvent — creating default organization",
			"account_id", e.AccountID,
		)

		if err := orgCreator.CreateDefaultOrg(ctx, e.AccountID); err != nil {
			return fmt.Errorf("create default org for account %s: %w", e.AccountID, err)
		}

		return nil
	})
}
