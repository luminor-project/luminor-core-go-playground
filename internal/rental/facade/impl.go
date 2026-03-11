package facade

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

type rentalQueryModel interface {
	FindByID(ctx context.Context, id string) (domain.Rental, error)
	FindBySubjectID(ctx context.Context, subjectID string) ([]domain.Rental, error)
	FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]domain.Rental, error)
	FindByOrgID(ctx context.Context, orgID string) ([]domain.Rental, error)
}

// Compile-time interface assertion.
var _ interface {
	CreateRental(ctx context.Context, dto CreateRentalDTO) (string, error)
	ListRentalsByOrg(ctx context.Context, orgID string) ([]RentalInfoDTO, error)
	ListRentalsByTenant(ctx context.Context, tenantPartyID string) ([]RentalInfoDTO, error)
	ListRentalsBySubject(ctx context.Context, subjectID string) ([]RentalInfoDTO, error)
} = (*facadeImpl)(nil)

type facadeImpl struct {
	store      eventstore.Store
	bus        *eventbus.Bus
	clock      domain.Clock
	checker    domain.DuplicateChecker
	queryModel rentalQueryModel
}

// New creates a new rental facade.
func New(store eventstore.Store, bus *eventbus.Bus, clock domain.Clock, checker domain.DuplicateChecker, queryModel rentalQueryModel) *facadeImpl {
	return &facadeImpl{store: store, bus: bus, clock: clock, checker: checker, queryModel: queryModel}
}

func (f *facadeImpl) CreateRental(ctx context.Context, dto CreateRentalDTO) (string, error) {
	rentalID := uuid.New().String()
	streamID := "rental-" + rentalID

	domainEvents, err := domain.EstablishNewRental(ctx, f.checker, f.clock, domain.EstablishRentalCmd{
		RentalID:           rentalID,
		SubjectID:          dto.SubjectID,
		TenantPartyID:      dto.TenantPartyID,
		OrgID:              dto.OrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
	if err != nil {
		return "", err
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, 0, uncommitted)
	if err != nil {
		return "", fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)
	return rentalID, nil
}

func (f *facadeImpl) ListRentalsByOrg(ctx context.Context, orgID string) ([]RentalInfoDTO, error) {
	rentals, err := f.queryModel.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(rentals), nil
}

func (f *facadeImpl) ListRentalsByTenant(ctx context.Context, tenantPartyID string) ([]RentalInfoDTO, error) {
	rentals, err := f.queryModel.FindByTenantPartyID(ctx, tenantPartyID)
	if err != nil {
		return nil, err
	}
	return toDTOs(rentals), nil
}

func (f *facadeImpl) ListRentalsBySubject(ctx context.Context, subjectID string) ([]RentalInfoDTO, error) {
	rentals, err := f.queryModel.FindBySubjectID(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	return toDTOs(rentals), nil
}

func toDTOs(rentals []domain.Rental) []RentalInfoDTO {
	result := make([]RentalInfoDTO, len(rentals))
	for i, r := range rentals {
		result[i] = RentalInfoDTO{
			ID:            r.ID,
			SubjectID:     r.SubjectID,
			TenantPartyID: r.TenantPartyID,
			OrgID:         r.OrgID,
		}
	}
	return result
}

// toUncommitted converts domain events to uncommitted events for the event store.
func toUncommitted(domainEvents []domain.DomainEvent) []eventstore.UncommittedEvent {
	uncommitted := make([]eventstore.UncommittedEvent, len(domainEvents))
	for i, de := range domainEvents {
		uncommitted[i] = eventstore.UncommittedEvent{
			EventType: de.EventType,
			Payload:   de.Payload,
		}
	}
	return uncommitted
}

// publishAll publishes stored events to the eventbus as facade event types.
func (f *facadeImpl) publishAll(ctx context.Context, stored []eventstore.StoredEvent) {
	for _, se := range stored {
		payload, err := domain.DeserializeEvent(se.EventType, se.Payload)
		if err != nil {
			slog.Error("failed to deserialize event for publishing", "event_type", se.EventType, "error", err)
			continue
		}

		var publishErr error
		switch se.EventType {
		case domain.EventRentalEstablished:
			e := payload.(domain.RentalEstablished)
			publishErr = eventbus.Publish(ctx, f.bus, RentalEstablishedEvent{
				RentalID:           e.RentalID,
				SubjectID:          e.SubjectID,
				TenantPartyID:      e.TenantPartyID,
				OrgID:              e.OrgID,
				CreatedByAccountID: e.CreatedByAccountID,
				EstablishedAt:      e.EstablishedAt,
			})
		}

		if publishErr != nil {
			slog.Error("failed to publish event", "event_type", se.EventType, "error", publishErr)
		}
	}
}
