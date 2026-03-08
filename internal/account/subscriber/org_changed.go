package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type activeOrgSetter interface {
	SetActiveOrganization(ctx context.Context, accountID, orgID string) error
}

// RegisterOrgChangedSubscriber subscribes to ActiveOrgChangedEvent
// and updates the account's currently active organization.
func RegisterOrgChangedSubscriber(bus *eventbus.Bus, orgSetter activeOrgSetter) {
	eventbus.Subscribe(bus, func(ctx context.Context, e orgfacade.ActiveOrgChangedEvent) error {
		slog.Info("handling ActiveOrgChangedEvent",
			"account_id", e.AffectedUserID,
			"org_id", e.OrganizationID,
		)

		if err := orgSetter.SetActiveOrganization(ctx, e.AffectedUserID, e.OrganizationID); err != nil {
			return fmt.Errorf("set active organization for account %s: %w", e.AffectedUserID, err)
		}

		return nil
	})
}
