package facade_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/party/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
)

var testClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// fakeStore is an in-memory event store for testing.
type fakeStore struct {
	streams map[string][]eventstore.StoredEvent
}

func newFakeStore() *fakeStore {
	return &fakeStore{streams: make(map[string][]eventstore.StoredEvent)}
}

func (s *fakeStore) Append(_ context.Context, streamID string, expectedVersion int, events []eventstore.UncommittedEvent) ([]eventstore.StoredEvent, error) {
	existing := s.streams[streamID]
	if len(existing) != expectedVersion {
		return nil, eventstore.ErrConcurrencyConflict
	}

	var stored []eventstore.StoredEvent
	for i, ue := range events {
		raw, err := json.Marshal(ue.Payload)
		if err != nil {
			return nil, err
		}
		stored = append(stored, eventstore.StoredEvent{
			ID:            "evt-" + streamID + "-" + string(rune('0'+i)),
			StreamID:      streamID,
			StreamVersion: expectedVersion + i + 1,
			EventType:     ue.EventType,
			Payload:       raw,
			RecordedAt:    time.Now(),
		})
	}
	s.streams[streamID] = append(existing, stored...)
	return stored, nil
}

func (s *fakeStore) LoadStream(_ context.Context, streamID string) ([]eventstore.StoredEvent, error) {
	return s.streams[streamID], nil
}

// fakeReadModel is an in-memory read model for testing.
type fakeReadModel struct {
	parties map[string]domain.Party
}

func newFakeReadModel() *fakeReadModel {
	return &fakeReadModel{parties: make(map[string]domain.Party)}
}

func (r *fakeReadModel) FindByID(_ context.Context, id string) (domain.Party, error) {
	p, ok := r.parties[id]
	if !ok {
		return domain.Party{}, domain.ErrPartyNotFound
	}
	return p, nil
}

func (r *fakeReadModel) FindByIDs(_ context.Context, ids []string) ([]domain.Party, error) {
	var result []domain.Party
	for _, id := range ids {
		if p, ok := r.parties[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *fakeReadModel) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range r.parties {
		if p.OwningOrganizationID == orgID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (r *fakeReadModel) FindByOrgAndKind(_ context.Context, orgID string, kind domain.PartyKind) ([]domain.Party, error) {
	var result []domain.Party
	for _, p := range r.parties {
		if p.OwningOrganizationID == orgID && p.PartyKind == kind {
			result = append(result, p)
		}
	}
	return result, nil
}

func TestCreateParty_AppendsEventsToStore(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

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

	streamID := "party-" + id
	events := store.streams[streamID]
	if len(events) != 1 {
		t.Fatalf("expected 1 event in stream, got %d", len(events))
	}
	if events[0].EventType != domain.EventPartyRegistered {
		t.Errorf("expected event type %s, got %s", domain.EventPartyRegistered, events[0].EventType)
	}
}

func TestCreateParty_PublishesEvent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()

	var published bool
	eventbus.Subscribe(bus, func(_ context.Context, e facade.PartyRegisteredEvent) error {
		published = true
		if e.Name != "Anna Schmidt" {
			t.Errorf("expected name 'Anna Schmidt', got %q", e.Name)
		}
		if e.ActorKind != facade.ActorKindHuman {
			t.Errorf("expected actor kind 'human', got %q", e.ActorKind)
		}
		if e.PartyKind != facade.PartyKindTenant {
			t.Errorf("expected party kind 'tenant', got %q", e.PartyKind)
		}
		if e.OrgID != "org-1" {
			t.Errorf("expected org 'org-1', got %q", e.OrgID)
		}
		return nil
	})

	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.CreateParty(context.Background(), facade.CreatePartyDTO{
		Name:               "Anna Schmidt",
		ActorKind:          facade.ActorKindHuman,
		PartyKind:          facade.PartyKindTenant,
		OwningOrgID:        "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !published {
		t.Error("expected PartyRegisteredEvent to be published")
	}
}

func TestCreateParty_InvalidKind_ReturnsError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.CreateParty(context.Background(), facade.CreatePartyDTO{
		Name:      "Test",
		ActorKind: facade.ActorKindHuman,
		PartyKind: facade.PartyKind("unknown"),
	})
	if err == nil {
		t.Fatal("expected error for invalid party kind")
	}
}

func TestGetPartyInfo_MapsCorrectly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna Schmidt",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

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
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.GetPartyInfo(context.Background(), "nonexistent")
	if !errors.Is(err, facade.ErrPartyNotFound) {
		t.Errorf("expected ErrPartyNotFound, got %v", err)
	}
}

func TestListPartiesByOrg_ReturnsMatching(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

	parties, err := fac.ListPartiesByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parties) != 1 {
		t.Errorf("expected 1 party, got %d", len(parties))
	}
}

func TestListPartiesByOrgAndKind_FiltersCorrectly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.parties["p-1"] = domain.Party{
		ID:                   "p-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
		Name:                 "Anna",
		OwningOrganizationID: "org-1",
	}
	readModel.parties["p-2"] = domain.Party{
		ID:                   "p-2",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindPropertyManager,
		Name:                 "Sarah",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

	tenants, err := fac.ListPartiesByOrgAndKind(context.Background(), "org-1", facade.PartyKindTenant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(tenants))
	}
}

func TestGetPartiesByIDs_ReturnsMatching(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.parties["p-1"] = domain.Party{
		ID: "p-1", Name: "Anna",
		OwningOrganizationID: "org-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindTenant,
	}
	readModel.parties["p-2"] = domain.Party{
		ID: "p-2", Name: "Sarah",
		OwningOrganizationID: "org-1",
		ActorKind:            domain.ActorKindHuman,
		PartyKind:            domain.PartyKindPropertyManager,
	}
	fac := facade.New(store, bus, testClock, readModel)

	parties, err := fac.GetPartiesByIDs(context.Background(), []string{"p-1", "p-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parties) != 2 {
		t.Errorf("expected 2 parties, got %d", len(parties))
	}
}
