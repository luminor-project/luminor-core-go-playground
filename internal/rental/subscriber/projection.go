package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type projectionWriter interface {
	UpsertProjection(ctx context.Context, id, subjectID, tenantPartyID, orgID, createdByAccountID string, createdAt time.Time) error
}

func RegisterProjectionSubscribers(bus *eventbus.Bus, writer projectionWriter) {
	eventbus.Subscribe(bus, func(ctx context.Context, e rentalfacade.RentalEstablishedEvent) error {
		slog.Info("projecting RentalEstablishedEvent", "rental_id", e.RentalID)
		if err := writer.UpsertProjection(ctx, e.RentalID, e.SubjectID, e.TenantPartyID, e.OrgID, e.CreatedByAccountID, e.EstablishedAt); err != nil {
			return fmt.Errorf("upsert rental projection: %w", err)
		}
		return nil
	})
}
