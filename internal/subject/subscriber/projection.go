package subscriber

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

type projectionWriter interface {
	UpsertProjection(ctx context.Context, id, subjectKind, name, detail, orgID, createdByAccountID string, createdAt time.Time) error
}

func RegisterProjectionSubscribers(bus *eventbus.Bus, writer projectionWriter) {
	eventbus.Subscribe(bus, func(ctx context.Context, e subjectfacade.SubjectRegisteredEvent) error {
		slog.Info("projecting SubjectRegisteredEvent", "subject_id", e.SubjectID)
		if err := writer.UpsertProjection(ctx, e.SubjectID, string(e.SubjectKind), e.Name, e.Detail, e.OrgID, e.CreatedByAccountID, e.RegisteredAt); err != nil {
			return fmt.Errorf("upsert subject projection: %w", err)
		}
		return nil
	})
}
