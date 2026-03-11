package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

func TestDeserializeEvent_RentalEstablished(t *testing.T) {
	t.Parallel()

	r := domain.NewRental(testClock)
	events, err := r.EstablishRental(domain.EstablishRentalCmd{
		RentalID:           "rental-1",
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
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

	got, err := domain.DeserializeEvent(domain.EventRentalEstablished, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	e, ok := got.(domain.RentalEstablished)
	if !ok {
		t.Fatalf("expected RentalEstablished, got %T", got)
	}
	if e.RentalID != "rental-1" {
		t.Errorf("expected rental ID 'rental-1', got %q", e.RentalID)
	}
	if e.SubjectID != "subject-1" {
		t.Errorf("expected subject ID 'subject-1', got %q", e.SubjectID)
	}
	if e.TenantPartyID != "party-1" {
		t.Errorf("expected tenant party ID 'party-1', got %q", e.TenantPartyID)
	}
	if e.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", e.OrgID)
	}
	if e.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", e.CreatedByAccountID)
	}
	if e.EstablishedAt != testClock.Now() {
		t.Errorf("expected established at %v, got %v", testClock.Now(), e.EstablishedAt)
	}
}

func TestDeserializeEvent_UnknownType(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent("rental.Unknown.v1", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestDeserializeEvent_MalformedJSON(t *testing.T) {
	t.Parallel()
	_, err := domain.DeserializeEvent(domain.EventRentalEstablished, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDeserializeEvent_Roundtrip_ApplyConsistency(t *testing.T) {
	t.Parallel()

	r1 := domain.NewRental(testClock)
	events, _ := r1.EstablishRental(domain.EstablishRentalCmd{
		RentalID:           "rental-1",
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	applyAll(r1, events)

	raw, _ := json.Marshal(events[0].Payload)
	deserialized, err := domain.DeserializeEvent(events[0].EventType, raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	r2 := domain.NewRental(testClock)
	r2.Apply(events[0].EventType, deserialized)

	if r1.ID != r2.ID {
		t.Errorf("ID mismatch: %q vs %q", r1.ID, r2.ID)
	}
	if r1.SubjectID != r2.SubjectID {
		t.Errorf("SubjectID mismatch: %q vs %q", r1.SubjectID, r2.SubjectID)
	}
	if r1.TenantPartyID != r2.TenantPartyID {
		t.Errorf("TenantPartyID mismatch: %q vs %q", r1.TenantPartyID, r2.TenantPartyID)
	}
	if r1.OrgID != r2.OrgID {
		t.Errorf("OrgID mismatch: %q vs %q", r1.OrgID, r2.OrgID)
	}
	if r1.CreatedByAccountID != r2.CreatedByAccountID {
		t.Errorf("CreatedByAccountID mismatch: %q vs %q", r1.CreatedByAccountID, r2.CreatedByAccountID)
	}
	if r1.Version != r2.Version {
		t.Errorf("Version mismatch: %d vs %d", r1.Version, r2.Version)
	}
}
