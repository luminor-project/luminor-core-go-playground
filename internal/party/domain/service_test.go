package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// mockRepository is an in-memory repository for testing.
type mockRepository struct {
	parties map[string]domain.Party
}

func newMockRepo() *mockRepository {
	return &mockRepository{parties: make(map[string]domain.Party)}
}

func (m *mockRepository) Create(_ context.Context, party domain.Party) error {
	m.parties[party.ID] = party
	return nil
}

func (m *mockRepository) FindByID(_ context.Context, id string) (domain.Party, error) {
	p, ok := m.parties[id]
	if !ok {
		return domain.Party{}, domain.ErrPartyNotFound
	}
	return p, nil
}

func (m *mockRepository) FindByIDs(_ context.Context, ids []string) ([]domain.Party, error) {
	var result []domain.Party
	for _, id := range ids {
		if p, ok := m.parties[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepository) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range m.parties {
		if p.OwningOrganizationID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepository) FindByOrgAndKind(_ context.Context, orgID string, kind domain.PartyKind) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range m.parties {
		if p.OwningOrganizationID == orgID && p.PartyKind == kind {
			result = append(result, p)
		}
	}
	return result, nil
}

func TestCreateParty_Success(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	party, err := svc.CreateParty(context.Background(), "Anna Schmidt", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if party.ID == "" {
		t.Error("expected non-empty ID")
	}
	if party.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", party.Name)
	}
	if party.ActorKind != domain.ActorKindHuman {
		t.Errorf("expected actor kind 'human', got %q", party.ActorKind)
	}
	if party.PartyKind != domain.PartyKindTenant {
		t.Errorf("expected party kind 'tenant', got %q", party.PartyKind)
	}
	if party.OwningOrganizationID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", party.OwningOrganizationID)
	}
	if party.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", party.CreatedByAccountID)
	}
	if party.CreatedAt != testClock.Now() {
		t.Errorf("expected created at %v, got %v", testClock.Now(), party.CreatedAt)
	}
}

func TestCreateParty_TrimsName(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	party, err := svc.CreateParty(context.Background(), "  Anna Schmidt  ", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if party.Name != "Anna Schmidt" {
		t.Errorf("expected trimmed name, got %q", party.Name)
	}
}

func TestCreateParty_EmptyName(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	_, err := svc.CreateParty(context.Background(), "   ", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	if !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestCreateParty_InvalidPartyKind(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	_, err := svc.CreateParty(context.Background(), "Test", domain.ActorKindHuman, domain.PartyKind("unknown"), "org-1", "account-1")
	if !errors.Is(err, domain.ErrInvalidPartyKind) {
		t.Errorf("expected ErrInvalidPartyKind, got %v", err)
	}
}

func TestCreateParty_PersistsToRepo(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	party, err := svc.CreateParty(context.Background(), "Anna Schmidt", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found, err := svc.FindByID(context.Background(), party.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", found.Name)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	_, err := svc.FindByID(context.Background(), "nonexistent")
	if !errors.Is(err, domain.ErrPartyNotFound) {
		t.Errorf("expected ErrPartyNotFound, got %v", err)
	}
}

func TestFindByOrganizationID(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	_, _ = svc.CreateParty(context.Background(), "Anna", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	_, _ = svc.CreateParty(context.Background(), "Sarah", domain.ActorKindHuman, domain.PartyKindPropertyManager, "org-1", "account-1")
	_, _ = svc.CreateParty(context.Background(), "Other", domain.ActorKindHuman, domain.PartyKindTenant, "org-2", "account-2")

	parties, err := svc.FindByOrganizationID(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parties) != 2 {
		t.Errorf("expected 2 parties for org-1, got %d", len(parties))
	}
}

func TestFindByOrgAndKind(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	_, _ = svc.CreateParty(context.Background(), "Anna", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	_, _ = svc.CreateParty(context.Background(), "Sarah", domain.ActorKindHuman, domain.PartyKindPropertyManager, "org-1", "account-1")

	tenants, err := svc.FindByOrgAndKind(context.Background(), "org-1", domain.PartyKindTenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(tenants))
	}
	if tenants[0].Name != "Anna" {
		t.Errorf("expected tenant name 'Anna', got %q", tenants[0].Name)
	}
}

func TestFindByIDs(t *testing.T) {
	t.Parallel()
	repo := newMockRepo()
	svc := domain.NewPartyService(repo, testClock)

	p1, _ := svc.CreateParty(context.Background(), "Anna", domain.ActorKindHuman, domain.PartyKindTenant, "org-1", "account-1")
	p2, _ := svc.CreateParty(context.Background(), "Sarah", domain.ActorKindHuman, domain.PartyKindPropertyManager, "org-1", "account-1")

	parties, err := svc.FindByIDs(context.Background(), []string{p1.ID, p2.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parties) != 2 {
		t.Errorf("expected 2 parties, got %d", len(parties))
	}
}
