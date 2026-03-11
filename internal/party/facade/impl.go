package facade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
)

type partyReader interface {
	FindByID(ctx context.Context, id string) (domain.Party, error)
	FindByIDs(ctx context.Context, ids []string) ([]domain.Party, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]domain.Party, error)
	FindByOrgAndKind(ctx context.Context, orgID string, kind domain.PartyKind) ([]domain.Party, error)
}

// Compile-time interface assertion.
var _ interface {
	GetPartyInfo(ctx context.Context, partyID string) (PartyInfoDTO, error)
	CreateParty(ctx context.Context, dto CreatePartyDTO) (string, error)
	ListPartiesByOrg(ctx context.Context, orgID string) ([]PartyInfoDTO, error)
	ListPartiesByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]PartyInfoDTO, error)
	GetPartiesByIDs(ctx context.Context, ids []string) ([]PartyInfoDTO, error)
} = (*facadeImpl)(nil)

type facadeImpl struct {
	store     eventstore.Store
	bus       *eventbus.Bus
	clock     domain.Clock
	readModel partyReader
}

// New creates a new party facade.
func New(store eventstore.Store, bus *eventbus.Bus, clock domain.Clock, readModel partyReader) *facadeImpl {
	return &facadeImpl{
		store:     store,
		bus:       bus,
		clock:     clock,
		readModel: readModel,
	}
}

// CreateParty registers a new party via event sourcing.
func (f *facadeImpl) CreateParty(ctx context.Context, dto CreatePartyDTO) (string, error) {
	partyID := uuid.New().String()
	streamID := "party-" + partyID

	p := domain.NewParty(f.clock)
	domainEvents, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:            partyID,
		Name:               dto.Name,
		ActorKind:          domain.ActorKind(dto.ActorKind),
		PartyKind:          domain.PartyKind(dto.PartyKind),
		OrgID:              dto.OwningOrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
	if err != nil {
		return "", fmt.Errorf("register party: %w", err)
	}

	uncommitted := toUncommitted(domainEvents)
	stored, err := f.store.Append(ctx, streamID, 0, uncommitted)
	if err != nil {
		return "", fmt.Errorf("append events: %w", err)
	}

	f.publishAll(ctx, stored)

	return partyID, nil
}

// GetPartyInfo returns a single party from the read model.
func (f *facadeImpl) GetPartyInfo(ctx context.Context, partyID string) (PartyInfoDTO, error) {
	p, err := f.readModel.FindByID(ctx, partyID)
	if err != nil {
		if errors.Is(err, domain.ErrPartyNotFound) {
			return PartyInfoDTO{}, ErrPartyNotFound
		}
		return PartyInfoDTO{}, err
	}
	return toDTO(p), nil
}

// ListPartiesByOrg returns all parties in an organization from the read model.
func (f *facadeImpl) ListPartiesByOrg(ctx context.Context, orgID string) ([]PartyInfoDTO, error) {
	parties, err := f.readModel.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(parties), nil
}

// ListPartiesByOrgAndKind returns parties of a specific kind in an organization from the read model.
func (f *facadeImpl) ListPartiesByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]PartyInfoDTO, error) {
	parties, err := f.readModel.FindByOrgAndKind(ctx, orgID, domain.PartyKind(kind))
	if err != nil {
		return nil, err
	}
	return toDTOs(parties), nil
}

// GetPartiesByIDs returns parties by their IDs from the read model.
func (f *facadeImpl) GetPartiesByIDs(ctx context.Context, ids []string) ([]PartyInfoDTO, error) {
	parties, err := f.readModel.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toDTOs(parties), nil
}

func toDTO(p domain.Party) PartyInfoDTO {
	return PartyInfoDTO{
		ID:        p.ID,
		ActorKind: ActorKind(p.ActorKind),
		PartyKind: PartyKind(p.PartyKind),
		Name:      p.Name,
	}
}

func toDTOs(parties []domain.Party) []PartyInfoDTO {
	result := make([]PartyInfoDTO, len(parties))
	for i, p := range parties {
		result[i] = toDTO(p)
	}
	return result
}

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

func (f *facadeImpl) publishAll(ctx context.Context, stored []eventstore.StoredEvent) {
	for _, se := range stored {
		payload, err := domain.DeserializeEvent(se.EventType, se.Payload)
		if err != nil {
			slog.Error("failed to deserialize event for publishing", "event_type", se.EventType, "error", err)
			continue
		}

		var publishErr error
		switch se.EventType {
		case domain.EventPartyRegistered:
			e := payload.(domain.PartyRegistered)
			publishErr = eventbus.Publish(ctx, f.bus, PartyRegisteredEvent{
				PartyID:            e.PartyID,
				ActorKind:          ActorKind(e.ActorKind),
				PartyKind:          PartyKind(e.PartyKind),
				Name:               e.Name,
				OrgID:              e.OrgID,
				CreatedByAccountID: e.CreatedByAccountID,
				RegisteredAt:       e.RegisteredAt,
			})
		}

		if publishErr != nil {
			slog.Error("failed to publish event", "event_type", se.EventType, "error", publishErr)
		}
	}
}
