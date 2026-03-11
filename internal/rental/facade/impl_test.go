package facade_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/rental/facade"
)

// fakeStore is an in-memory event store for testing.
type fakeStore struct {
	streams map[string][]eventstore.StoredEvent
}

func newFakeStore() *fakeStore {
	return &fakeStore{streams: make(map[string][]eventstore.StoredEvent)}
}

func (s *fakeStore) Append(_ context.Context, streamID string, expectedVersion int, events []eventstore.UncommittedEvent) ([]eventstore.StoredEvent, error) {
	current := s.streams[streamID]
	if len(current) != expectedVersion {
		return nil, eventstore.ErrConcurrencyConflict
	}

	var stored []eventstore.StoredEvent
	for i, ue := range events {
		payload, err := json.Marshal(ue.Payload)
		if err != nil {
			return nil, err
		}
		stored = append(stored, eventstore.StoredEvent{
			ID:            streamID + "-" + string(rune('0'+expectedVersion+i)),
			StreamID:      streamID,
			StreamVersion: expectedVersion + i + 1,
			EventType:     ue.EventType,
			Payload:       payload,
		})
	}

	s.streams[streamID] = append(current, stored...)
	return stored, nil
}

func (s *fakeStore) LoadStream(_ context.Context, streamID string) ([]eventstore.StoredEvent, error) {
	return s.streams[streamID], nil
}

// fakeChecker implements domain.DuplicateChecker for testing.
type fakeChecker struct {
	rentals map[string]domain.Rental
}

func (c *fakeChecker) ExistsBySubjectAndTenant(_ context.Context, subjectID, tenantPartyID string) (bool, error) {
	for _, rental := range c.rentals {
		if rental.SubjectID == subjectID && rental.TenantPartyID == tenantPartyID {
			return true, nil
		}
	}
	return false, nil
}

// fakeQueryModel is an in-memory read model for testing query methods.
type fakeQueryModel struct {
	rentals map[string]domain.Rental
}

func newFakes() (*fakeChecker, *fakeQueryModel) {
	rentals := make(map[string]domain.Rental)
	return &fakeChecker{rentals: rentals}, &fakeQueryModel{rentals: rentals}
}

func (r *fakeQueryModel) FindByID(_ context.Context, id string) (domain.Rental, error) {
	rental, ok := r.rentals[id]
	if !ok {
		return domain.Rental{}, domain.ErrRentalNotFound
	}
	return rental, nil
}

func (r *fakeQueryModel) FindBySubjectID(_ context.Context, subjectID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, rental := range r.rentals {
		if rental.SubjectID == subjectID {
			result = append(result, rental)
		}
	}
	return result, nil
}

func (r *fakeQueryModel) FindByTenantPartyID(_ context.Context, tenantPartyID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, rental := range r.rentals {
		if rental.TenantPartyID == tenantPartyID {
			result = append(result, rental)
		}
	}
	return result, nil
}

func (r *fakeQueryModel) FindByOrgID(_ context.Context, orgID string) ([]domain.Rental, error) {
	var result []domain.Rental
	for _, rental := range r.rentals {
		if rental.OrgID == orgID {
			result = append(result, rental)
		}
	}
	return result, nil
}

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

func TestCreateRental_Success(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()
	fac := facade.New(store, bus, testClock, checker, queryModel)

	id, err := fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}

	// Verify event was stored.
	streamID := "rental-" + id
	events := store.streams[streamID]
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].EventType != domain.EventRentalEstablished {
		t.Errorf("expected event type %s, got %s", domain.EventRentalEstablished, events[0].EventType)
	}
}

func TestCreateRental_Duplicate(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()

	// Pre-populate with an existing rental (shared backing map).
	checker.rentals["existing"] = domain.Rental{
		ID:            "existing",
		SubjectID:     "s-1",
		TenantPartyID: "p-1",
		OrgID:         "org-1",
	}

	fac := facade.New(store, bus, testClock, checker, queryModel)

	_, err := fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID:          "s-1",
		TenantPartyID:      "p-1",
		OrgID:              "org-1",
		CreatedByAccountID: "a-1",
	})
	if !errors.Is(err, facade.ErrDuplicateRental) {
		t.Errorf("expected ErrDuplicateRental, got %v", err)
	}
}

func TestListRentalsByOrg(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()
	queryModel.rentals["r-1"] = domain.Rental{ID: "r-1", SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1"}

	fac := facade.New(store, bus, testClock, checker, queryModel)

	rentals, err := fac.ListRentalsByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental, got %d", len(rentals))
	}
}

func TestListRentalsByTenant(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()
	queryModel.rentals["r-1"] = domain.Rental{ID: "r-1", SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1"}

	fac := facade.New(store, bus, testClock, checker, queryModel)

	rentals, err := fac.ListRentalsByTenant(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental, got %d", len(rentals))
	}
}

func TestListRentalsBySubject(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()
	queryModel.rentals["r-1"] = domain.Rental{ID: "r-1", SubjectID: "s-1", TenantPartyID: "p-1", OrgID: "org-1"}

	fac := facade.New(store, bus, testClock, checker, queryModel)

	rentals, err := fac.ListRentalsBySubject(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rentals) != 1 {
		t.Errorf("expected 1 rental, got %d", len(rentals))
	}
}

func TestCreateRental_PublishesEvent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	checker, queryModel := newFakes()
	fac := facade.New(store, bus, testClock, checker, queryModel)

	var received facade.RentalEstablishedEvent
	eventbus.Subscribe(bus, func(_ context.Context, e facade.RentalEstablishedEvent) error {
		received = e
		return nil
	})

	id, err := fac.CreateRental(context.Background(), facade.CreateRentalDTO{
		SubjectID:          "subject-1",
		TenantPartyID:      "party-1",
		OrgID:              "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.RentalID != id {
		t.Errorf("expected published rental ID %q, got %q", id, received.RentalID)
	}
	if received.SubjectID != "subject-1" {
		t.Errorf("expected published subject ID 'subject-1', got %q", received.SubjectID)
	}
	if received.TenantPartyID != "party-1" {
		t.Errorf("expected published tenant party ID 'party-1', got %q", received.TenantPartyID)
	}
	if received.OrgID != "org-1" {
		t.Errorf("expected published org ID 'org-1', got %q", received.OrgID)
	}
	if received.CreatedByAccountID != "account-1" {
		t.Errorf("expected published created by 'account-1', got %q", received.CreatedByAccountID)
	}
	if received.EstablishedAt != testClock.Now() {
		t.Errorf("expected published established at %v, got %v", testClock.Now(), received.EstablishedAt)
	}
}
