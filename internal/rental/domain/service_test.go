package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
)

// fakeDuplicateChecker implements domain.DuplicateChecker for testing.
type fakeDuplicateChecker struct {
	exists bool
	err    error
}

func (f *fakeDuplicateChecker) ExistsBySubjectAndTenant(_ context.Context, _, _ string) (bool, error) {
	return f.exists, f.err
}

func TestEstablishNewRental_Success(t *testing.T) {
	t.Parallel()
	checker := &fakeDuplicateChecker{exists: false}

	events, err := domain.EstablishNewRental(context.Background(), checker, testClock, domain.EstablishRentalCmd{
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
}

func TestEstablishNewRental_Duplicate(t *testing.T) {
	t.Parallel()
	checker := &fakeDuplicateChecker{exists: true}

	_, err := domain.EstablishNewRental(context.Background(), checker, testClock, domain.EstablishRentalCmd{
		RentalID:           "rental-1",
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if !errors.Is(err, domain.ErrDuplicateRental) {
		t.Errorf("expected ErrDuplicateRental, got %v", err)
	}
}

func TestEstablishNewRental_CheckerError(t *testing.T) {
	t.Parallel()
	checkerErr := errors.New("db connection failed")
	checker := &fakeDuplicateChecker{err: checkerErr}

	_, err := domain.EstablishNewRental(context.Background(), checker, testClock, domain.EstablishRentalCmd{
		RentalID:           "rental-1",
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, checkerErr) {
		t.Errorf("expected wrapped checker error, got %v", err)
	}
}
