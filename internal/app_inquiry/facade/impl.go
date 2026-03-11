package facade

import (
	"context"
	"fmt"

	casehandlingfacade "github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/facade"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

type rentalLookup interface {
	ListRentalsByTenant(ctx context.Context, tenantPartyID string) ([]rentalfacade.RentalInfoDTO, error)
}

type caseHandler interface {
	HandleInboundInquiry(ctx context.Context, dto casehandlingfacade.InquiryDTO) (string, error)
}

type partyLookup interface {
	ListPartiesByOrgAndKind(ctx context.Context, orgID string, kind partyfacade.PartyKind) ([]partyfacade.PartyInfoDTO, error)
}

// Compile-time interface assertion.
var _ interface {
	SubmitInquiry(ctx context.Context, dto SubmitInquiryDTO) (string, error)
} = (*facadeImpl)(nil)

type facadeImpl struct {
	rentals rentalLookup
	cases   caseHandler
	parties partyLookup
}

// New creates a new inquiry facade.
func New(rentals rentalLookup, cases caseHandler, parties partyLookup) *facadeImpl {
	return &facadeImpl{
		rentals: rentals,
		cases:   cases,
		parties: parties,
	}
}

// SubmitInquiry resolves the tenant's rental (to find the subject), finds the
// operator and agent parties, then creates a work item via case handling.
func (f *facadeImpl) SubmitInquiry(ctx context.Context, dto SubmitInquiryDTO) (string, error) {
	// Find the tenant's rental to resolve the subject.
	rentals, err := f.rentals.ListRentalsByTenant(ctx, dto.TenantPartyID)
	if err != nil {
		return "", fmt.Errorf("list rentals for tenant %s: %w", dto.TenantPartyID, err)
	}
	if len(rentals) == 0 {
		return "", fmt.Errorf("no rental found for tenant %s", dto.TenantPartyID)
	}

	// Use the first rental's subject (simplification — could allow selection later).
	rental := rentals[0]

	// Find the operator (property_manager) and agent (assistant) in this org.
	operatorPartyID := ""
	agentPartyID := ""

	pms, err := f.parties.ListPartiesByOrgAndKind(ctx, dto.OrgID, partyfacade.PartyKindPropertyManager)
	if err == nil && len(pms) > 0 {
		operatorPartyID = pms[0].ID
	}

	agents, err := f.parties.ListPartiesByOrgAndKind(ctx, dto.OrgID, partyfacade.PartyKindAssistant)
	if err == nil && len(agents) > 0 {
		agentPartyID = agents[0].ID
	}

	return f.cases.HandleInboundInquiry(ctx, casehandlingfacade.InquiryDTO{
		SenderPartyID:   dto.TenantPartyID,
		OperatorPartyID: operatorPartyID,
		AgentPartyID:    agentPartyID,
		SubjectID:       rental.SubjectID,
		Body:            dto.Body,
	})
}
