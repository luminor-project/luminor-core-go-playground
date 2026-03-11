package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

func applyAll(p *domain.Party, events []domain.DomainEvent) {
	for _, e := range events {
		p.Apply(e.EventType, e.Payload)
	}
}

func TestRegisterParty_Success(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	events, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:            "party-1",
		Name:               "Anna Schmidt",
		ActorKind:          domain.ActorKindHuman,
		PartyKind:          domain.PartyKindTenant,
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventPartyRegistered {
		t.Errorf("expected event type %s, got %s", domain.EventPartyRegistered, events[0].EventType)
	}

	payload := events[0].Payload.(domain.PartyRegistered)
	if payload.PartyID != "party-1" {
		t.Errorf("expected party ID 'party-1', got %q", payload.PartyID)
	}
	if payload.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", payload.Name)
	}
	if payload.ActorKind != domain.ActorKindHuman {
		t.Errorf("expected actor kind 'human', got %q", payload.ActorKind)
	}
	if payload.PartyKind != domain.PartyKindTenant {
		t.Errorf("expected party kind 'tenant', got %q", payload.PartyKind)
	}
	if payload.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", payload.OrgID)
	}
	if payload.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", payload.CreatedByAccountID)
	}
	if payload.RegisteredAt != testClock.Now() {
		t.Errorf("expected registered at %v, got %v", testClock.Now(), payload.RegisteredAt)
	}
}

func TestRegisterParty_TrimsName(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	events, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:   "party-1",
		Name:      "  Anna Schmidt  ",
		ActorKind: domain.ActorKindHuman,
		PartyKind: domain.PartyKindTenant,
		OrgID:     "org-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := events[0].Payload.(domain.PartyRegistered)
	if payload.Name != "Anna Schmidt" {
		t.Errorf("expected trimmed name, got %q", payload.Name)
	}
}

func TestRegisterParty_EmptyName(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	_, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:   "party-1",
		Name:      "   ",
		ActorKind: domain.ActorKindHuman,
		PartyKind: domain.PartyKindTenant,
		OrgID:     "org-1",
	})
	if !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestRegisterParty_InvalidPartyKind(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	_, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:   "party-1",
		Name:      "Test",
		ActorKind: domain.ActorKindHuman,
		PartyKind: domain.PartyKind("unknown"),
		OrgID:     "org-1",
	})
	if !errors.Is(err, domain.ErrInvalidPartyKind) {
		t.Errorf("expected ErrInvalidPartyKind, got %v", err)
	}
}

func TestRegisterParty_AlreadyRegistered(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	events, _ := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:   "party-1",
		Name:      "Anna Schmidt",
		ActorKind: domain.ActorKindHuman,
		PartyKind: domain.PartyKindTenant,
		OrgID:     "org-1",
	})
	applyAll(p, events)

	_, err := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:   "party-2",
		Name:      "Someone Else",
		ActorKind: domain.ActorKindHuman,
		PartyKind: domain.PartyKindTenant,
		OrgID:     "org-1",
	})
	if !errors.Is(err, domain.ErrAlreadyRegistered) {
		t.Errorf("expected ErrAlreadyRegistered, got %v", err)
	}
}

func TestApply_PartyRegistered(t *testing.T) {
	t.Parallel()
	p := domain.NewParty(testClock)

	events, _ := p.RegisterParty(domain.RegisterPartyCmd{
		PartyID:            "party-1",
		Name:               "Anna Schmidt",
		ActorKind:          domain.ActorKindHuman,
		PartyKind:          domain.PartyKindTenant,
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	applyAll(p, events)

	if p.ID != "party-1" {
		t.Errorf("expected ID 'party-1', got %q", p.ID)
	}
	if p.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", p.Name)
	}
	if p.ActorKind != domain.ActorKindHuman {
		t.Errorf("expected actor kind 'human', got %q", p.ActorKind)
	}
	if p.PartyKind != domain.PartyKindTenant {
		t.Errorf("expected party kind 'tenant', got %q", p.PartyKind)
	}
	if p.OwningOrganizationID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", p.OwningOrganizationID)
	}
	if p.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", p.CreatedByAccountID)
	}
	if !p.Registered {
		t.Error("expected Registered=true")
	}
	if p.Version != 1 {
		t.Errorf("expected Version 1, got %d", p.Version)
	}
}

func TestValidPartyKinds(t *testing.T) {
	t.Parallel()
	kinds := domain.ValidPartyKinds()
	if len(kinds) != 3 {
		t.Errorf("expected 3 valid party kinds, got %d", len(kinds))
	}
}

func TestIsValidPartyKind(t *testing.T) {
	t.Parallel()
	if !domain.IsValidPartyKind(domain.PartyKindTenant) {
		t.Error("expected tenant to be valid")
	}
	if !domain.IsValidPartyKind(domain.PartyKindPropertyManager) {
		t.Error("expected property_manager to be valid")
	}
	if !domain.IsValidPartyKind(domain.PartyKindAssistant) {
		t.Error("expected assistant to be valid")
	}
	if domain.IsValidPartyKind(domain.PartyKind("bogus")) {
		t.Error("expected 'bogus' to be invalid")
	}
}
