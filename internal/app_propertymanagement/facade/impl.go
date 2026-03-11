package facade

import (
	"context"
	"fmt"

	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

type partyCreator interface {
	CreateParty(ctx context.Context, dto partyfacade.CreatePartyDTO) (string, error)
}

type subjectCreator interface {
	CreateSubject(ctx context.Context, dto subjectfacade.CreateSubjectDTO) (string, error)
}

type rentalCreator interface {
	CreateRental(ctx context.Context, dto rentalfacade.CreateRentalDTO) (string, error)
}

type pendingLinkCreator interface {
	CreatePendingPartyLink(ctx context.Context, invitationID, partyID, orgID string) (string, error)
}

type invitationCreator interface {
	CreateInvitationAsActor(ctx context.Context, actorUserID, orgID, email string) (orgfacade.InvitationDTO, error)
}

// Compile-time interface assertion.
var _ interface {
	CreateProperty(ctx context.Context, dto CreatePropertyDTO) (string, error)
	CreateTenant(ctx context.Context, dto CreateTenantDTO) (string, error)
	AssignTenantToProperty(ctx context.Context, dto AssignTenantDTO) (string, error)
	InviteTenant(ctx context.Context, dto InviteTenantDTO) error
} = (*facadeImpl)(nil)

type facadeImpl struct {
	parties     partyCreator
	subjects    subjectCreator
	rentals     rentalCreator
	accounts    pendingLinkCreator
	invitations invitationCreator
}

// New creates a new property management facade.
func New(parties partyCreator, subjects subjectCreator, rentals rentalCreator, accounts pendingLinkCreator, invitations invitationCreator) *facadeImpl {
	return &facadeImpl{
		parties:     parties,
		subjects:    subjects,
		rentals:     rentals,
		accounts:    accounts,
		invitations: invitations,
	}
}

func (f *facadeImpl) CreateProperty(ctx context.Context, dto CreatePropertyDTO) (string, error) {
	return f.subjects.CreateSubject(ctx, subjectfacade.CreateSubjectDTO{
		Name:               dto.Name,
		Detail:             dto.Detail,
		OwningOrgID:        dto.OrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
}

func (f *facadeImpl) CreateTenant(ctx context.Context, dto CreateTenantDTO) (string, error) {
	return f.parties.CreateParty(ctx, partyfacade.CreatePartyDTO{
		Name:               dto.Name,
		ActorKind:          partyfacade.ActorKindHuman,
		PartyKind:          partyfacade.PartyKindTenant,
		OwningOrgID:        dto.OrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
}

func (f *facadeImpl) AssignTenantToProperty(ctx context.Context, dto AssignTenantDTO) (string, error) {
	return f.rentals.CreateRental(ctx, rentalfacade.CreateRentalDTO{
		SubjectID:          dto.SubjectID,
		TenantPartyID:      dto.TenantPartyID,
		OrgID:              dto.OrgID,
		CreatedByAccountID: dto.CreatedByAccountID,
	})
}

func (f *facadeImpl) InviteTenant(ctx context.Context, dto InviteTenantDTO) error {
	inv, err := f.invitations.CreateInvitationAsActor(ctx, dto.ActorAccountID, dto.OrgID, dto.Email)
	if err != nil {
		return fmt.Errorf("create invitation for tenant: %w", err)
	}

	_, err = f.accounts.CreatePendingPartyLink(ctx, inv.ID, dto.TenantPartyID, dto.OrgID)
	if err != nil {
		return fmt.Errorf("create pending party link for tenant invitation: %w", err)
	}

	return nil
}
