package facade_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
)

// fakeService implements the service interface consumed by the facade.
type fakeService struct {
	parties map[string]domain.Party
}

func newFakeService() *fakeService {
	return &fakeService{parties: make(map[string]domain.Party)}
}

func (f *fakeService) CreateParty(_ context.Context, name string, actorKind domain.ActorKind, partyKind domain.PartyKind, orgID, createdByAccountID string) (domain.Party, error) {
	p := domain.Party{
		ID:                   "generated-id",
		ActorKind:            actorKind,
		PartyKind:            partyKind,
		Name:                 name,
		OwningOrganizationID: orgID,
		CreatedByAccountID:   createdByAccountID,
		CreatedAt:            time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}
	f.parties[p.ID] = p
	return p, nil
}

func (f *fakeService) FindByID(_ context.Context, id string) (domain.Party, error) {
	p, ok := f.parties[id]
	if !ok {
		return domain.Party{}, domain.ErrPartyNotFound
	}
	return p, nil
}

func (f *fakeService) FindByIDs(_ context.Context, ids []string) ([]domain.Party, error) {
	var result []domain.Party
	for _, id := range ids {
		if p, ok := f.parties[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (f *fakeService) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range f.parties {
		if p.OwningOrganizationID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (f *fakeService) FindByOrgAndKind(_ context.Context, orgID string, kind domain.PartyKind) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range f.parties {
		if p.OwningOrganizationID == orgID && p.PartyKind == kind {
			result = append(result, p)
		}
	}
	return result, nil
}

func TestGetPartyInfo_MapsCorrectly(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna Schmidt",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(svc)

	dto, err := fac.GetPartyInfo(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.ID != "p-1" {
		t.Errorf("expected ID 'p-1', got %q", dto.ID)
	}
	if dto.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", dto.Name)
	}
	if dto.ActorKind != facade.ActorKindHuman {
		t.Errorf("expected actor kind 'human', got %q", dto.ActorKind)
	}
	if dto.PartyKind != facade.PartyKindTenant {
		t.Errorf("expected party kind 'tenant', got %q", dto.PartyKind)
	}
}

func TestGetPartyInfo_NotFound(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	_, err := fac.GetPartyInfo(context.Background(), "nonexistent")
	if !errors.Is(err, facade.ErrPartyNotFound) {
		t.Errorf("expected ErrPartyNotFound, got %v", err)
	}
}

func TestCreateParty_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	fac := facade.New(svc)

	id, err := fac.CreateParty(context.Background(), facade.CreatePartyDTO{
		Name:               "Anna Schmidt",
		ActorKind:          facade.ActorKindHuman,
		PartyKind:          facade.PartyKindTenant,
		OwningOrgID:        "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestListPartiesByOrg_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(svc)

	parties, err := fac.ListPartiesByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parties) != 1 {
		t.Errorf("expected 1 party, got %d", len(parties))
	}
}

func TestListPartiesByOrgAndKind_DelegatesToService(t *testing.T) {
	t.Parallel()
	svc := newFakeService()
	svc.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna",
		OwningOrganizationID: "org-1",
	}
	svc.parties["p-2"] = domain.Party{
		ID:                   "p-2",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindPropertyManager,
		Name:                 "Sarah",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(svc)

	tenants, err := fac.ListPartiesByOrgAndKind(context.Background(), "org-1", facade.PartyKindTenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(tenants))
	}
}
