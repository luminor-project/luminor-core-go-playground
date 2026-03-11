package domain

import (
	"context"
	"fmt"
)

// Repository defines the persistence interface for rentals.
type Repository interface {
	Create(ctx context.Context, rental Rental) error
	FindByID(ctx context.Context, id string) (Rental, error)
	FindBySubjectID(ctx context.Context, subjectID string) ([]Rental, error)
	FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]Rental, error)
	FindByOrgID(ctx context.Context, orgID string) ([]Rental, error)
	ExistsBySubjectAndTenant(ctx context.Context, subjectID, tenantPartyID string) (bool, error)
}

// RentalService handles rental business logic.
type RentalService struct {
	repo  Repository
	clock Clock
}

// NewRentalService creates a new RentalService.
func NewRentalService(repo Repository, clock Clock) *RentalService {
	return &RentalService{repo: repo, clock: clock}
}

// CreateRental validates uniqueness and persists a new rental.
func (s *RentalService) CreateRental(ctx context.Context, subjectID, tenantPartyID, orgID, createdByAccountID string) (Rental, error) {
	exists, err := s.repo.ExistsBySubjectAndTenant(ctx, subjectID, tenantPartyID)
	if err != nil {
		return Rental{}, fmt.Errorf("check duplicate rental: %w", err)
	}
	if exists {
		return Rental{}, ErrDuplicateRental
	}

	rental := NewRental(subjectID, tenantPartyID, orgID, createdByAccountID, s.clock.Now())

	if err := s.repo.Create(ctx, rental); err != nil {
		return Rental{}, fmt.Errorf("create rental: %w", err)
	}

	return rental, nil
}

// FindByTenantPartyID returns all rentals for a tenant.
func (s *RentalService) FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]Rental, error) {
	return s.repo.FindByTenantPartyID(ctx, tenantPartyID)
}

// FindBySubjectID returns all rentals for a subject.
func (s *RentalService) FindBySubjectID(ctx context.Context, subjectID string) ([]Rental, error) {
	return s.repo.FindBySubjectID(ctx, subjectID)
}

// FindByOrgID returns all rentals in an organization.
func (s *RentalService) FindByOrgID(ctx context.Context, orgID string) ([]Rental, error) {
	return s.repo.FindByOrgID(ctx, orgID)
}
