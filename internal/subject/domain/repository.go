package domain

import "context"

// Repository defines the read model interface for subjects.
type Repository interface {
	FindByID(ctx context.Context, id string) (Subject, error)
	FindByIDs(ctx context.Context, ids []string) ([]Subject, error)
	FindByOrganizationID(ctx context.Context, orgID string) ([]Subject, error)
	FindByOrgAndKind(ctx context.Context, orgID string, kind SubjectKind) ([]Subject, error)
}
