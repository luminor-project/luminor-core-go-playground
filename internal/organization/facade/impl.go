package facade

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/luminor-project/luminor-core-go-playground/internal/organization/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type orgService interface {
	GetOrganizationName(ctx context.Context, orgID string) (string, error)
	GetAllOrganizationsForUser(ctx context.Context, userID string) ([]domain.Organization, error)
	CreateOrganization(ctx context.Context, ownerID, name string) (domain.Organization, error)
}

// Compile-time interface assertion: facadeImpl satisfies all consumer interfaces.
var _ interface {
	GetOrganizationNameByID(ctx context.Context, orgID string) (string, error)
	CreateDefaultOrg(ctx context.Context, accountID string) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	service orgService
	bus     *eventbus.Bus
}

// New creates a new organization facade implementation.
func New(service orgService, bus *eventbus.Bus) *facadeImpl {
	return &facadeImpl{
		service: service,
		bus:     bus,
	}
}

func (f *facadeImpl) GetOrganizationNameByID(ctx context.Context, orgID string) (string, error) {
	return f.service.GetOrganizationName(ctx, orgID)
}

func (f *facadeImpl) CreateDefaultOrg(ctx context.Context, accountID string) error {
	slog.Info("creating default organization", "account_id", accountID)

	existing, err := f.service.GetAllOrganizationsForUser(ctx, accountID)
	if err != nil {
		return fmt.Errorf("check existing organizations: %w", err)
	}
	if len(existing) > 0 {
		slog.Info("default organization already exists; skipping", "account_id", accountID)
		return nil
	}

	org, err := f.service.CreateOrganization(ctx, accountID, "")
	if err != nil {
		return fmt.Errorf("create default organization: %w", err)
	}

	// Dispatch event to set active organization
	if err := eventbus.Publish(ctx, f.bus, ActiveOrgChangedEvent{
		OrganizationID: org.ID,
		AffectedUserID: accountID,
	}); err != nil {
		return fmt.Errorf("publish ActiveOrgChangedEvent: %w", err)
	}

	return nil
}
