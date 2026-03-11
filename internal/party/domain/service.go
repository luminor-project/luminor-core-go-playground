package domain

import (
	"context"
	"fmt"
	"time"
)

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}

// Repository defines the persistence interface for parties.
type Repository interface {
	Create(ctx context.Context, party Party) error
	FindByID(ctx context.Context, id string) (Party, error)
	FindByIDs(ctx context.Context, ids []string) ([]Party, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]Party, error)
	FindByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]Party, error)
}

// PartyService handles party business logic.
type PartyService struct {
	repo  Repository
	clock Clock
}

// NewPartyService creates a new PartyService.
func NewPartyService(repo Repository, clock Clock) *PartyService {
	return &PartyService{repo: repo, clock: clock}
}

// CreateParty validates inputs and persists a new party.
func (s *PartyService) CreateParty(ctx context.Context, name string, actorKind ActorKind, partyKind PartyKind, orgID, createdByAccountID string) (Party, error) {
	party, err := NewParty(name, actorKind, partyKind, orgID, createdByAccountID, s.clock.Now())
	if err != nil {
		return Party{}, err
	}

	if err := s.repo.Create(ctx, party); err != nil {
		return Party{}, fmt.Errorf("create party: %w", err)
	}

	return party, nil
}

// FindByID returns a party by ID.
func (s *PartyService) FindByID(ctx context.Context, id string) (Party, error) {
	return s.repo.FindByID(ctx, id)
}

// FindByIDs returns parties by IDs.
func (s *PartyService) FindByIDs(ctx context.Context, ids []string) ([]Party, error) {
	return s.repo.FindByIDs(ctx, ids)
}

// FindByOrganizationID returns all parties belonging to an organization.
func (s *PartyService) FindByOrganizationID(ctx context.Context, orgID string) ([]Party, error) {
	return s.repo.FindByOrganizationID(ctx, orgID)
}

// FindByOrgAndKind returns parties of a specific kind in an organization.
func (s *PartyService) FindByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]Party, error) {
	return s.repo.FindByOrgAndKind(ctx, orgID, kind)
}
