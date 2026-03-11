package domain_test

import (
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

func applyAll(r *domain.Rental, events []domain.DomainEvent) {
	for _, e := range events {
		r.Apply(e.EventType, e.Payload)
	}
}

func TestEstablishRental_Success(t *testing.T) {
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
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventRentalEstablished {
		t.Errorf("expected event type %s, got %s", domain.EventRentalEstablished, events[0].EventType)
	}

	payload := events[0].Payload.(domain.RentalEstablished)
	if payload.RentalID != "rental-1" {
		t.Errorf("expected rental ID 'rental-1', got %q", payload.RentalID)
	}
	if payload.SubjectID != "subject-1" {
		t.Errorf("expected subject ID 'subject-1', got %q", payload.SubjectID)
	}
	if payload.TenantPartyID != "party-1" {
		t.Errorf("expected tenant party ID 'party-1', got %q", payload.TenantPartyID)
	}
	if payload.OrgID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %q", payload.OrgID)
	}
	if payload.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", payload.CreatedByAccountID)
	}
	if payload.EstablishedAt != testClock.Now() {
		t.Errorf("expected established at %v, got %v", testClock.Now(), payload.EstablishedAt)
	}
}

func TestEstablishRental_AlreadyEstablished(t *testing.T) {
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
		t.Fatalf("first establish failed: %v", err)
	}
	applyAll(r, events)

	_, err = r.EstablishRental(domain.EstablishRentalCmd{
		RentalID:           "rental-2",
		SubjectID:          "subject-2",
		TenantPartyID:      "party-2",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != domain.ErrAlreadyEstablished {
		t.Errorf("expected ErrAlreadyEstablished, got %v", err)
	}
}

func TestApply_RentalEstablished(t *testing.T) {
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
	applyAll(r, events)

	if r.ID != "rental-1" {
		t.Errorf("expected ID 'rental-1', got %q", r.ID)
	}
	if r.SubjectID != "subject-1" {
		t.Errorf("expected SubjectID 'subject-1', got %q", r.SubjectID)
	}
	if r.TenantPartyID != "party-1" {
		t.Errorf("expected TenantPartyID 'party-1', got %q", r.TenantPartyID)
	}
	if r.OrgID != "org-1" {
		t.Errorf("expected OrgID 'org-1', got %q", r.OrgID)
	}
	if r.CreatedByAccountID != "account-1" {
		t.Errorf("expected CreatedByAccountID 'account-1', got %q", r.CreatedByAccountID)
	}
	if !r.Established {
		t.Error("expected Established to be true")
	}
	if r.Version != 1 {
		t.Errorf("expected Version 1, got %d", r.Version)
	}
}
