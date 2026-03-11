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
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/domain"
	"github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
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
	subjects map[string]domain.Subject
}

func newFakeReadModel() *fakeReadModel {
	return &fakeReadModel{subjects: make(map[string]domain.Subject)}
}

func (r *fakeReadModel) FindByID(_ context.Context, id string) (domain.Subject, error) {
	s, ok := r.subjects[id]
	if !ok {
		return domain.Subject{}, domain.ErrSubjectNotFound
	}
	return s, nil
}

func (r *fakeReadModel) FindByIDs(_ context.Context, ids []string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, id := range ids {
		if s, ok := r.subjects[id]; ok {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *fakeReadModel) FindByOrganizationID(_ context.Context, orgID string) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, s := range r.subjects {
		if s.OwningOrganizationID == orgID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *fakeReadModel) FindByOrgAndKind(_ context.Context, orgID string, kind domain.SubjectKind) ([]domain.Subject, error) {
	var result []domain.Subject
	for _, s := range r.subjects {
		if s.OwningOrganizationID == orgID && s.SubjectKind == kind {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestCreateSubject_AppendsEventsToStore(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

	id, err := fac.CreateSubject(context.Background(), facade.CreateSubjectDTO{
		SubjectKind:        facade.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OwningOrgID:        "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}

	streamID := "subject-" + id
	events := store.streams[streamID]
	if len(events) != 1 {
		t.Fatalf("expected 1 event in stream, got %d", len(events))
	}
	if events[0].EventType != domain.EventSubjectRegistered {
		t.Errorf("expected event type %s, got %s", domain.EventSubjectRegistered, events[0].EventType)
	}
}

func TestCreateSubject_PublishesEvent(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()

	var published bool
	eventbus.Subscribe(bus, func(_ context.Context, e facade.SubjectRegisteredEvent) error {
		published = true
		if e.Name != "Flussufer Apartments" {
			t.Errorf("expected name 'Flussufer Apartments', got %q", e.Name)
		}
		if e.SubjectKind != facade.SubjectKindDwelling {
			t.Errorf("expected kind 'dwelling', got %q", e.SubjectKind)
		}
		if e.Detail != "Unit 12A" {
			t.Errorf("expected detail 'Unit 12A', got %q", e.Detail)
		}
		if e.OrgID != "org-1" {
			t.Errorf("expected org 'org-1', got %q", e.OrgID)
		}
		return nil
	})

	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.CreateSubject(context.Background(), facade.CreateSubjectDTO{
		SubjectKind:        facade.SubjectKindDwelling,
		Name:               "Flussufer Apartments",
		Detail:             "Unit 12A",
		OwningOrgID:        "org-1",
		CreatedByAccountID: "account-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !published {
		t.Error("expected SubjectRegisteredEvent to be published")
	}
}

func TestCreateSubject_InvalidKind_ReturnsError(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.CreateSubject(context.Background(), facade.CreateSubjectDTO{
		SubjectKind: facade.SubjectKind("unknown"),
		Name:        "Test",
	})
	if err == nil {
		t.Fatal("expected error for invalid subject kind")
	}
}

func TestGetSubjectInfo_MapsCorrectly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.subjects["s-1"] = domain.Subject{
		ID:                   "s-1",
		SubjectKind:          domain.SubjectKindDwelling,
		Name:                 "Flussufer Apartments",
		Detail:               "Unit 12A",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

	dto, err := fac.GetSubjectInfo(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.ID != "s-1" {
		t.Errorf("expected ID 's-1', got %q", dto.ID)
	}
	if dto.SubjectKind != facade.SubjectKindDwelling {
		t.Errorf("expected kind 'dwelling', got %q", dto.SubjectKind)
	}
	if dto.Name != "Flussufer Apartments" {
		t.Errorf("expected name 'Flussufer Apartments', got %q", dto.Name)
	}
	if dto.Detail != "Unit 12A" {
		t.Errorf("expected detail 'Unit 12A', got %q", dto.Detail)
	}
}

func TestGetSubjectInfo_NotFound(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	fac := facade.New(store, bus, testClock, readModel)

	_, err := fac.GetSubjectInfo(context.Background(), "nonexistent")
	if !errors.Is(err, facade.ErrSubjectNotFound) {
		t.Errorf("expected ErrSubjectNotFound, got %v", err)
	}
}

func TestListSubjectsByOrg_ReturnsMatching(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.subjects["s-1"] = domain.Subject{
		ID:                   "s-1",
		SubjectKind:          domain.SubjectKindDwelling,
		Name:                 "Property A",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

	subjects, err := fac.ListSubjectsByOrg(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subjects) != 1 {
		t.Errorf("expected 1 subject, got %d", len(subjects))
	}
}

func TestListSubjectsByOrgAndKind_FiltersCorrectly(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.subjects["s-1"] = domain.Subject{
		ID:                   "s-1",
		SubjectKind:          domain.SubjectKindDwelling,
		Name:                 "Dwelling A",
		OwningOrganizationID: "org-1",
	}
	fac := facade.New(store, bus, testClock, readModel)

	subjects, err := fac.ListSubjectsByOrgAndKind(context.Background(), "org-1", facade.SubjectKindDwelling)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subjects) != 1 {
		t.Errorf("expected 1 dwelling, got %d", len(subjects))
	}
	if subjects[0].SubjectKind != facade.SubjectKindDwelling {
		t.Errorf("expected kind 'dwelling', got %q", subjects[0].SubjectKind)
	}
}

func TestGetSubjectsByIDs_ReturnsMatching(t *testing.T) {
	t.Parallel()
	store := newFakeStore()
	bus := eventbus.New()
	readModel := newFakeReadModel()
	readModel.subjects["s-1"] = domain.Subject{
		ID:          "s-1",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "Property A",
	}
	readModel.subjects["s-2"] = domain.Subject{
		ID:          "s-2",
		SubjectKind: domain.SubjectKindDwelling,
		Name:        "Property B",
	}
	fac := facade.New(store, bus, testClock, readModel)

	subjects, err := fac.GetSubjectsByIDs(context.Background(), []string{"s-1", "s-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subjects) != 2 {
		t.Errorf("expected 2 subjects, got %d", len(subjects))
	}
}
