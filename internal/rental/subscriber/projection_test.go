package subscriber_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	rentalfacade "github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/subscriber"
)

type upsertCall struct {
	ID                 string
	SubjectID          string
	TenantPartyID      string
	OrgID              string
	CreatedByAccountID string
	CreatedAt          time.Time
}

type fakeWriter struct {
	calls []upsertCall
	err   error
}

func (w *fakeWriter) UpsertProjection(_ context.Context, id, subjectID, tenantPartyID, orgID, createdByAccountID string, createdAt time.Time) error {
	w.calls = append(w.calls, upsertCall{
		ID:                 id,
		SubjectID:          subjectID,
		TenantPartyID:      tenantPartyID,
		OrgID:              orgID,
		CreatedByAccountID: createdByAccountID,
		CreatedAt:          createdAt,
	})
	return w.err
}

func TestProjection_RentalEstablished(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	err := eventbus.Publish(context.Background(), bus, rentalfacade.RentalEstablishedEvent{
		RentalID:           "rental-1",
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
		EstablishedAt:      now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.calls) != 1 {
		t.Fatalf("expected 1 upsert call, got %d", len(writer.calls))
	}

	c := writer.calls[0]
	if c.ID != "rental-1" {
		t.Errorf("expected ID 'rental-1', got %q", c.ID)
	}
	if c.SubjectID != "subject-1" {
		t.Errorf("expected subject ID 'subject-1', got %q", c.SubjectID)
	}
	if c.TenantPartyID != "party-1" {
		t.Errorf("expected tenant party ID 'party-1', got %q", c.TenantPartyID)
	}
	if c.OrgID != "org-1" {
		t.Errorf("expected org 'org-1', got %q", c.OrgID)
	}
	if c.CreatedByAccountID != "account-1" {
		t.Errorf("expected created by 'account-1', got %q", c.CreatedByAccountID)
	}
	if c.CreatedAt != now {
		t.Errorf("expected created at %v, got %v", now, c.CreatedAt)
	}
}

func TestProjection_RentalEstablished_WriterError(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	writer := &fakeWriter{err: fmt.Errorf("db is down")}
	subscriber.RegisterProjectionSubscribers(bus, writer)

	err := eventbus.Publish(context.Background(), bus, rentalfacade.RentalEstablishedEvent{
		RentalID:      "rental-1",
		SubjectID:     "subject-1",
		TenantPartyID: "party-1",
		OrgID:         "org-1",
		EstablishedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error when writer fails")
	}
}
