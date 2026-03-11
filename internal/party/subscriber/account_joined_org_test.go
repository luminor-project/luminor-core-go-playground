package subscriber

import (
	"context"
	"testing"

	accountfacade "github.com/luminor-project/luminor-core-go-playground/internal/account/facade"
	orgfacade "github.com/luminor-project/luminor-core-go-playground/internal/organization/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type fakePartyFacade struct {
	createPartyCalls int
	lastCreatedDTO   partyfacade.CreatePartyDTO
	createdPartyID   string
}

func (f *fakePartyFacade) CreateParty(_ context.Context, dto partyfacade.CreatePartyDTO) (string, error) {
	f.createPartyCalls++
	f.lastCreatedDTO = dto
	return f.createdPartyID, nil
}

type fakeAccountFacade struct {
	linkPartyCalls    int
	lastLinkedPartyID string

	memberships []accountfacade.PartyMembershipDTO
}

func (f *fakeAccountFacade) LinkPartyToAccount(_ context.Context, _, partyID, _ string) error {
	f.linkPartyCalls++
	f.lastLinkedPartyID = partyID
	return nil
}

func (f *fakeAccountFacade) GetPartyMembershipsForAccount(_ context.Context, _, _ string) ([]accountfacade.PartyMembershipDTO, error) {
	return f.memberships, nil
}

func (f *fakeAccountFacade) SetActiveParty(_ context.Context, _, _ string) error {
	return nil
}

func (f *fakeAccountFacade) GetAccountInfoByID(_ context.Context, _ string) (accountfacade.AccountInfoDTO, error) {
	return accountfacade.AccountInfoDTO{Email: "user@example.com"}, nil
}

func TestAccountJoinedOrg_CreatesPMPartyAndLinks(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	partyFac := &fakePartyFacade{createdPartyID: "new-party-1"}
	acctFac := &fakeAccountFacade{memberships: nil} // no existing memberships

	RegisterAccountJoinedOrgSubscriber(bus, partyFac, acctFac)

	err := eventbus.Publish(context.Background(), bus, orgfacade.AccountJoinedOrgEvent{
		AccountID: "acct-1",
		OrgID:     "org-1",
		Email:     "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if partyFac.createPartyCalls != 1 {
		t.Fatalf("expected CreateParty called once, got %d", partyFac.createPartyCalls)
	}
	if partyFac.lastCreatedDTO.PartyKind != partyfacade.PartyKindPropertyManager {
		t.Errorf("expected PartyKindPropertyManager, got %q", partyFac.lastCreatedDTO.PartyKind)
	}
	if partyFac.lastCreatedDTO.ActorKind != partyfacade.ActorKindHuman {
		t.Errorf("expected ActorKindHuman, got %q", partyFac.lastCreatedDTO.ActorKind)
	}
	if acctFac.linkPartyCalls != 1 {
		t.Fatalf("expected LinkPartyToAccount called once, got %d", acctFac.linkPartyCalls)
	}
	if acctFac.lastLinkedPartyID != "new-party-1" {
		t.Errorf("expected linked party 'new-party-1', got %q", acctFac.lastLinkedPartyID)
	}
}

func TestAccountJoinedOrg_SkipsIfPartyAlreadyExists(t *testing.T) {
	t.Parallel()

	bus := eventbus.New()
	partyFac := &fakePartyFacade{createdPartyID: "new-party-1"}
	acctFac := &fakeAccountFacade{
		memberships: []accountfacade.PartyMembershipDTO{
			{AccountID: "acct-1", PartyID: "existing-party", OrgID: "org-1"},
		},
	}

	RegisterAccountJoinedOrgSubscriber(bus, partyFac, acctFac)

	err := eventbus.Publish(context.Background(), bus, orgfacade.AccountJoinedOrgEvent{
		AccountID: "acct-1",
		OrgID:     "org-1",
		Email:     "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if partyFac.createPartyCalls != 0 {
		t.Fatalf("expected CreateParty NOT called, got %d", partyFac.createPartyCalls)
	}
	if acctFac.linkPartyCalls != 0 {
		t.Fatalf("expected LinkPartyToAccount NOT called, got %d", acctFac.linkPartyCalls)
	}
}
