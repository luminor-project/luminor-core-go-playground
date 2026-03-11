package domain

import "context"

// Repository defines the read model interface for rentals.
type Repository interface {
	FindByID(ctx context.Context, id string) (Rental, error)
	FindBySubjectID(ctx context.Context, subjectID string) ([]Rental, error)
	FindByTenantPartyID(ctx context.Context, tenantPartyID string) ([]Rental, error)
	FindByOrgID(ctx context.Context, orgID string) ([]Rental, error)
	ExistsBySubjectAndTenant(ctx context.Context, subjectID, tenantPartyID string) (bool, error)
}
