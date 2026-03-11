package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type projectionWriter interface {
	UpsertProjection(ctx context.Context, id, actorKind, partyKind, name, orgID, createdByAccountID string, createdAt time.Time) error
}

// RegisterProjectionSubscribers registers eventbus subscribers that project party events
// into the parties read model.
func RegisterProjectionSubscribers(bus *eventbus.Bus, writer projectionWriter) {
	eventbus.Subscribe(bus, func(ctx context.Context, e partyfacade.PartyRegisteredEvent) error {
		slog.Info("projecting PartyRegisteredEvent", "party_id", e.PartyID)
		if err := writer.UpsertProjection(ctx, e.PartyID, string(e.ActorKind), string(e.PartyKind), e.Name, e.OrgID, e.CreatedByAccountID, e.RegisteredAt); err != nil {
			return fmt.Errorf("upsert party projection %s: %w", e.PartyID, err)
		}
		return nil
	})
}
