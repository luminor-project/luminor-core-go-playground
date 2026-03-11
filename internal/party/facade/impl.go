package facade

import (
	"context"
	"errors"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
)

type partyService interface {
	CreateParty(ctx context.Context, name string, actorKind domain.ActorKind, partyKind domain.PartyKind, orgID, createdByAccountID string) (domain.Party, error)
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
	service partyService
}

// New creates a new party facade.
func New(service partyService) *facadeImpl {
	return &facadeImpl{service: service}
}

func (f *facadeImpl) GetPartyInfo(ctx context.Context, partyID string) (PartyInfoDTO, error) {
	p, err := f.service.FindByID(ctx, partyID)
	if err != nil {
		if errors.Is(err, domain.ErrPartyNotFound) {
			return PartyInfoDTO{}, ErrPartyNotFound
		}
		return PartyInfoDTO{}, err
	}
	return toDTO(p), nil
}

func (f *facadeImpl) CreateParty(ctx context.Context, dto CreatePartyDTO) (string, error) {
	p, err := f.service.CreateParty(ctx, dto.Name, domain.ActorKind(dto.ActorKind), domain.PartyKind(dto.PartyKind), dto.OwningOrgID, dto.CreatedByAccountID)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

func (f *facadeImpl) ListPartiesByOrg(ctx context.Context, orgID string) ([]PartyInfoDTO, error) {
	parties, err := f.service.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(parties), nil
}

func (f *facadeImpl) ListPartiesByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]PartyInfoDTO, error) {
	parties, err := f.service.FindByOrgAndKind(ctx, orgID, domain.PartyKind(kind))
	if err != nil {
		return nil, err
	}
	return toDTOs(parties), nil
}

func (f *facadeImpl) GetPartiesByIDs(ctx context.Context, ids []string) ([]PartyInfoDTO, error) {
	parties, err := f.service.FindByIDs(ctx, ids)
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
