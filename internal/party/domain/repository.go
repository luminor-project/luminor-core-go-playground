package domain

import "context"

// Repository defines the read model interface for parties.
type Repository interface {
	FindByID(ctx context.Context, id string) (Party, error)
	FindByIDs(ctx context.Context, ids []string) ([]Party, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]Party, error)
	FindByOrgAndKind(ctx context.Context, orgID string, kind PartyKind) ([]Party, error)
}
