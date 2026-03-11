package facade

import (
	"context"
	"errors"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

type rentalService interface {
	CreateRental(ctx context.Context, subjectID, tenantPartyID, orgID, createdByAccountID string) (domain.Rental, error)
	FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]domain.Rental, error)
	FindBySubjectID(ctx context.Context, subjectID string) ([]domain.Rental, error)
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
	service rentalService
}

// New creates a new rental facade.
func New(service rentalService) *facadeImpl {
	return &facadeImpl{service: service}
}

func (f *facadeImpl) CreateRental(ctx context.Context, dto CreateRentalDTO) (string, error) {
	r, err := f.service.CreateRental(ctx, dto.SubjectID, dto.TenantPartyID, dto.OrgID, dto.CreatedByAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateRental) {
			return "", ErrDuplicateRental
		}
		return "", err
	}
	return r.ID, nil
}

func (f *facadeImpl) ListRentalsByOrg(ctx context.Context, orgID string) ([]RentalInfoDTO, error) {
	rentals, err := f.service.FindByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return toDTOs(rentals), nil
}

func (f *facadeImpl) ListRentalsByTenant(ctx context.Context, tenantPartyID string) ([]RentalInfoDTO, error) {
	rentals, err := f.service.FindByTenantPartyID(ctx, tenantPartyID)
	if err != nil {
		return nil, err
	}
	return toDTOs(rentals), nil
}

func (f *facadeImpl) ListRentalsBySubject(ctx context.Context, subjectID string) ([]RentalInfoDTO, error) {
	rentals, err := f.service.FindBySubjectID(ctx, subjectID)
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
