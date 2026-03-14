package facade

import (
	"context"
	"testing"

	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
)

type fakePartyFacade struct {
	createPartyCalls int
	lastDTO          partyfacade.CreatePartyDTO
}

func (f *fakePartyFacade) CreateParty(_ context.Context, dto partyfacade.CreatePartyDTO) (string, error) {
	f.createPartyCalls++
	f.lastDTO = dto
	return "tenant-party-1", nil
}

type fakeSubjectFacade struct {
	createSubjectCalls int
	lastDTO            subjectfacade.CreateSubjectDTO
}

func (f *fakeSubjectFacade) CreateSubject(_ context.Context, dto subjectfacade.CreateSubjectDTO) (string, error) {
	f.createSubjectCalls++
	f.lastDTO = dto
	return "subject-1", nil
}

type fakeRentalFacade struct {
	createRentalCalls int
	lastDTO           rentalfacade.CreateRentalDTO
}

func (f *fakeRentalFacade) CreateRental(_ context.Context, dto rentalfacade.CreateRentalDTO) (string, error) {
	f.createRentalCalls++
	f.lastDTO = dto
	return "rental-1", nil
}

type fakeAccountFacade struct {
	createPendingLinkCalls int
	lastInvitationID       string
	lastPartyID            string
}

func (f *fakeAccountFacade) CreatePendingPartyLink(_ context.Context, invitationID, partyID, orgID string) (string, error) {
	f.createPendingLinkCalls++
	f.lastInvitationID = invitationID
	f.lastPartyID = partyID
	return "link-1", nil
}

type fakeOrgFacade struct {
	createInvitationCalls int
	lastEmail             string
}

func (f *fakeOrgFacade) CreateInvitationAsActor(_ context.Context, actorUserID, orgID, email string) (orgfacade.InvitationDTO, error) {
	f.createInvitationCalls++
	f.lastEmail = email
	return orgfacade.InvitationDTO{ID: "inv-1"}, nil
}

func TestCreateProperty_CallsSubjectFacade(t *testing.T) {
	t.Parallel()

	subjectFac := &fakeSubjectFacade{}
	fac := New(nil, subjectFac, nil, nil, nil)

	id, err := fac.CreateProperty(context.Background(), CreatePropertyDTO{
		Name:               "Flussufer Apartments, Unit 12A",
		Detail:             "3-room flat, ground floor",
		OrgID:              "org-1",
		CreatedByAccountID: "acct-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "subject-1" {
		t.Errorf("expected subject-1, got %q", id)
	}
	if subjectFac.createSubjectCalls != 1 {
		t.Fatalf("expected 1 call, got %d", subjectFac.createSubjectCalls)
	}
	if subjectFac.lastDTO.Name != "Flussufer Apartments, Unit 12A" {
		t.Errorf("expected name passed through, got %q", subjectFac.lastDTO.Name)
	}
	if subjectFac.lastDTO.SubjectKind != subjectfacade.SubjectKindDwelling {
		t.Errorf("expected SubjectKindDwelling, got %q", subjectFac.lastDTO.SubjectKind)
	}
}

func TestCreateTenant_CallsPartyFacadeWithTenantKind(t *testing.T) {
	t.Parallel()

	partyFac := &fakePartyFacade{}
	fac := New(partyFac, nil, nil, nil, nil)

	id, err := fac.CreateTenant(context.Background(), CreateTenantDTO{
		Name:               "Anna Schmidt",
		OrgID:              "org-1",
		CreatedByAccountID: "acct-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "tenant-party-1" {
		t.Errorf("expected tenant-party-1, got %q", id)
	}
	if partyFac.lastDTO.PartyKind != partyfacade.PartyKindTenant {
		t.Errorf("expected PartyKindTenant, got %q", partyFac.lastDTO.PartyKind)
	}
	if partyFac.lastDTO.ActorKind != partyfacade.ActorKindHuman {
		t.Errorf("expected ActorKindHuman, got %q", partyFac.lastDTO.ActorKind)
	}
}

func TestAssignTenantToProperty_CallsRentalFacade(t *testing.T) {
	t.Parallel()

	rentalFac := &fakeRentalFacade{}
	fac := New(nil, nil, rentalFac, nil, nil)

	id, err := fac.AssignTenantToProperty(context.Background(), AssignTenantDTO{
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "acct-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "rental-1" {
		t.Errorf("expected rental-1, got %q", id)
	}
	if rentalFac.lastDTO.SubjectID != "subject-1" {
		t.Errorf("expected subject-1, got %q", rentalFac.lastDTO.SubjectID)
	}
	if rentalFac.lastDTO.TenantPartyID != "party-1" {
		t.Errorf("expected party-1, got %q", rentalFac.lastDTO.TenantPartyID)
	}
}

func TestInviteTenant_CreatesInvitationAndPendingLink(t *testing.T) {
	t.Parallel()

	acctFac := &fakeAccountFacade{}
	orgFac := &fakeOrgFacade{}
	fac := New(nil, nil, nil, acctFac, orgFac)

	err := fac.InviteTenant(context.Background(), InviteTenantDTO{
		TenantPartyID:  "party-1",
		Email:          "anna@example.com",
		OrgID:          "org-1",
		ActorAccountID: "acct-pm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgFac.createInvitationCalls != 1 {
		t.Fatalf("expected 1 invitation call, got %d", orgFac.createInvitationCalls)
	}
	if orgFac.lastEmail != "anna@example.com" {
		t.Errorf("expected anna@example.com, got %q", orgFac.lastEmail)
	}
	if acctFac.createPendingLinkCalls != 1 {
		t.Fatalf("expected 1 pending link call, got %d", acctFac.createPendingLinkCalls)
	}
	if acctFac.lastPartyID != "party-1" {
		t.Errorf("expected party-1, got %q", acctFac.lastPartyID)
	}
	if acctFac.lastInvitationID != "inv-1" {
		t.Errorf("expected inv-1, got %q", acctFac.lastInvitationID)
	}
}

func TestCreatePropertyOwner_CallsPartyFacadeWithPropertyOwnerKind(t *testing.T) {
	t.Parallel()

	partyFac := &fakePartyFacade{}
	fac := New(partyFac, nil, nil, nil, nil)

	id, err := fac.CreatePropertyOwner(context.Background(), CreatePropertyOwnerDTO{
		Name:               "John Smith",
		OrgID:              "org-1",
		CreatedByAccountID: "acct-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "tenant-party-1" {
		t.Errorf("expected tenant-party-1, got %q", id)
	}
	if partyFac.lastDTO.PartyKind != partyfacade.PartyKindPropertyOwner {
		t.Errorf("expected PartyKindPropertyOwner, got %q", partyFac.lastDTO.PartyKind)
	}
	if partyFac.lastDTO.ActorKind != partyfacade.ActorKindHuman {
		t.Errorf("expected ActorKindHuman, got %q", partyFac.lastDTO.ActorKind)
	}
}

func TestInvitePropertyOwner_CreatesInvitationAndPendingLink(t *testing.T) {
	t.Parallel()

	acctFac := &fakeAccountFacade{}
	orgFac := &fakeOrgFacade{}
	fac := New(nil, nil, nil, acctFac, orgFac)

	err := fac.InvitePropertyOwner(context.Background(), InvitePropertyOwnerDTO{
		PropertyOwnerPartyID: "party-1",
		Email:                "john@example.com",
		OrgID:                "org-1",
		ActorAccountID:       "acct-pm",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgFac.createInvitationCalls != 1 {
		t.Fatalf("expected 1 invitation call, got %d", orgFac.createInvitationCalls)
	}
	if orgFac.lastEmail != "john@example.com" {
		t.Errorf("expected john@example.com, got %q", orgFac.lastEmail)
	}
	if acctFac.createPendingLinkCalls != 1 {
		t.Fatalf("expected 1 pending link call, got %d", acctFac.createPendingLinkCalls)
	}
	if acctFac.lastPartyID != "party-1" {
		t.Errorf("expected party-1, got %q", acctFac.lastPartyID)
	}
	if acctFac.lastInvitationID != "inv-1" {
		t.Errorf("expected inv-1, got %q", acctFac.lastInvitationID)
	}
}
