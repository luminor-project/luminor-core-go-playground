package facade

import (
	"context"
	"testing"

	casehandlingfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type fakeRentalFacade struct {
	rentals []rentalfacade.RentalInfoDTO
}

func (f *fakeRentalFacade) ListRentalsByTenant(_ context.Context, _ string) ([]rentalfacade.RentalInfoDTO, error) {
	return f.rentals, nil
}

type fakeCaseHandlingFacade struct {
	handleInquiryCalls int
	lastDTO            casehandlingfacade.InquiryDTO
}

func (f *fakeCaseHandlingFacade) HandleInboundInquiry(_ context.Context, dto casehandlingfacade.InquiryDTO) (string, error) {
	f.handleInquiryCalls++
	f.lastDTO = dto
	return "workitem-1", nil
}

type fakePartyLookup struct {
	parties map[string]partyfacade.PartyInfoDTO
}

func (f *fakePartyLookup) ListPartiesByOrgAndKind(_ context.Context, _ string, kind partyfacade.PartyKind) ([]partyfacade.PartyInfoDTO, error) {
	var result []partyfacade.PartyInfoDTO
	for _, p := range f.parties {
		if p.PartyKind == kind {
			result = append(result, p)
		}
	}
	return result, nil
}

func TestSubmitInquiry_ResolvesRentalAndCreatesWorkItem(t *testing.T) {
	t.Parallel()

	rentalFac := &fakeRentalFacade{
		rentals: []rentalfacade.RentalInfoDTO{
			{ID: "rental-1", SubjectID: "subject-1", TenantPartyID: "tenant-1", OrgID: "org-1"},
		},
	}
	caseFac := &fakeCaseHandlingFacade{}
	partyFac := &fakePartyLookup{
		parties: map[string]partyfacade.PartyInfoDTO{
			"pm-1":    {ID: "pm-1", PartyKind: partyfacade.PartyKindPropertyManager},
			"agent-1": {ID: "agent-1", PartyKind: partyfacade.PartyKindAssistant},
		},
	}

	fac := New(rentalFac, caseFac, partyFac)

	workItemID, err := fac.SubmitInquiry(context.Background(), SubmitInquiryDTO{
		TenantPartyID: "tenant-1",
		OrgID:         "org-1",
		Body:          "My heating is broken",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workItemID != "workitem-1" {
		t.Errorf("expected workitem-1, got %q", workItemID)
	}
	if caseFac.handleInquiryCalls != 1 {
		t.Fatalf("expected 1 call, got %d", caseFac.handleInquiryCalls)
	}
	if caseFac.lastDTO.SenderPartyID != "tenant-1" {
		t.Errorf("expected sender tenant-1, got %q", caseFac.lastDTO.SenderPartyID)
	}
	if caseFac.lastDTO.SubjectID != "subject-1" {
		t.Errorf("expected subject-1, got %q", caseFac.lastDTO.SubjectID)
	}
	if caseFac.lastDTO.Body != "My heating is broken" {
		t.Errorf("expected body preserved, got %q", caseFac.lastDTO.Body)
	}
}
