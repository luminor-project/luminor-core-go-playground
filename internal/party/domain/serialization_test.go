package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
)

func TestDeserializeEvent_PartyRegistered(t *testing.T) {
	t.Parallel()

	// Create a domain event via the aggregate, then round-trip through JSON.
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

	raw, err := json.Marshal(events[0].Payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := domain.DeserializeEvent(domain.EventPartyRegistered, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	e, ok := got.(domain.PartyRegistered)
	if !ok {
		t.Fatalf("expected PartyRegistered, got %T", got)
	}
	if e.PartyID != "party-1" {
		t.Errorf("expected party ID 'party-1', got %q", e.PartyID)
	}
	if e.Name != "Anna Schmidt" {
		t.Errorf("expected name 'Anna Schmidt', got %q", e.Name)
	}
	if e.ActorKind != domain.ActorKindHuman {
		t.Errorf("expected actor kind 'human', got %q", e.ActorKind)
	}
	if e.PartyKind != domain.PartyKindTenant {
		t.Errorf("expected party kind 'tenant', got %q", e.PartyKind)
	}
	if e.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", e.OrgID)
	}
	if e.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", e.CreatedByAccountID)
	}
	if e.RegisteredAt != testClock.Now() {
		t.Errorf("expected registered at %v, got %v", testClock.Now(), e.RegisteredAt)
	}
}

func TestDeserializeEvent_UnknownType(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent("party.Unknown.v1", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestDeserializeEvent_MalformedJSON(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent(domain.EventPartyRegistered, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDeserializeEvent_Roundtrip_ApplyConsistency(t *testing.T) {
	t.Parallel()

	// Verify that deserializing a stored event and applying it produces
	// the same aggregate state as applying the original domain event.
	p1 := domain.NewParty(testClock)
	events, _ := p1.RegisterParty(domain.RegisterPartyCmd{
		PartyID:            "party-1",
		Name:               "Anna Schmidt",
		ActorKind:          domain.ActorKindHuman,
		PartyKind:          domain.PartyKindTenant,
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})

	// Apply original events.
	applyAll(p1, events)

	// Simulate event store: marshal → unmarshal → apply.
	raw, _ := json.Marshal(events[0].Payload)
	deserialized, err := domain.DeserializeEvent(events[0].EventType, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	p2 := domain.NewParty(testClock)
	p2.Apply(events[0].EventType, deserialized)

	if p1.ID != p2.ID {
		t.Errorf("ID mismatch: %q vs %q", p1.ID, p2.ID)
	}
	if p1.Name != p2.Name {
		t.Errorf("Name mismatch: %q vs %q", p1.Name, p2.Name)
	}
	if p1.PartyKind != p2.PartyKind {
		t.Errorf("PartyKind mismatch: %q vs %q", p1.PartyKind, p2.PartyKind)
	}
	if p1.ActorKind != p2.ActorKind {
		t.Errorf("ActorKind mismatch: %q vs %q", p1.ActorKind, p2.ActorKind)
	}
	if p1.OwningOrganizationID != p2.OwningOrganizationID {
		t.Errorf("OwningOrganizationID mismatch: %q vs %q", p1.OwningOrganizationID, p2.OwningOrganizationID)
	}
	if p1.Version != p2.Version {
		t.Errorf("Version mismatch: %d vs %d", p1.Version, p2.Version)
	}
}
