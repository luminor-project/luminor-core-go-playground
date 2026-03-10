package subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/infra"
	"github.com/luminor-project/luminor-core-go-playground/internal/app_casehandling/subscriber"
	partyfacade "github.com/luminor-project/luminor-core-go-playground/internal/party/facade"
	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
	subjectfacade "github.com/luminor-project/luminor-core-go-playground/internal/subject/facade"
	workitemfacade "github.com/luminor-project/luminor-core-go-playground/internal/workitem/facade"
)

type fakeStore struct {
	upsertCalls         []infra.CaseDashboardRow
	appendTimelineCalls []appendTimelineCall
	updateStatusCalls   []updateStatusCall
}

type appendTimelineCall struct {
	WorkItemID string
	Entry      infra.TimelineEntry
}

type updateStatusCall struct {
	WorkItemID string
	Status     string
}

func (s *fakeStore) Upsert(_ context.Context, row infra.CaseDashboardRow) error {
	s.upsertCalls = append(s.upsertCalls, row)
	return nil
}

func (s *fakeStore) AppendTimeline(_ context.Context, workItemID string, entry infra.TimelineEntry) error {
	s.appendTimelineCalls = append(s.appendTimelineCalls, appendTimelineCall{workItemID, entry})
	return nil
}

func (s *fakeStore) UpdateStatus(_ context.Context, workItemID, status string) error {
	s.updateStatusCalls = append(s.updateStatusCalls, updateStatusCall{workItemID, status})
	return nil
}

func (s *fakeStore) AddNoteToTimeline(_ context.Context, _ string, _ int, _ infra.TimelineNote) error {
	return nil
}

func (s *fakeStore) EditNoteOnTimeline(_ context.Context, _, _, _ string, _ time.Time) error {
	return nil
}

func (s *fakeStore) DeleteNoteOnTimeline(_ context.Context, _, _ string) error {
	return nil
}

type fakePartyLookup struct {
	data map[string]partyfacade.PartyInfoDTO
}

func (f *fakePartyLookup) GetPartyInfo(_ context.Context, partyID string) (partyfacade.PartyInfoDTO, error) {
	p, ok := f.data[partyID]
	if !ok {
		return partyfacade.PartyInfoDTO{Name: partyID, ActorKind: partyfacade.ActorKind("unknown")}, nil
	}
	return p, nil
}

type fakeSubjectLookup struct {
	data map[string]subjectfacade.SubjectInfoDTO
}

func (f *fakeSubjectLookup) GetSubjectInfo(_ context.Context, subjectID string) (subjectfacade.SubjectInfoDTO, error) {
	s, ok := f.data[subjectID]
	if !ok {
		return subjectfacade.SubjectInfoDTO{Name: subjectID}, nil
	}
	return s, nil
}

func TestProjection_WorkItemCreated(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	store := &fakeStore{}
	parties := &fakePartyLookup{data: map[string]partyfacade.PartyInfoDTO{}}
	subjects := &fakeSubjectLookup{data: map[string]subjectfacade.SubjectInfoDTO{}}

	subscriber.RegisterProjectionSubscribers(bus, store, parties, subjects)

	now := time.Now()
	err := eventbus.Publish(context.Background(), bus, workitemfacade.WorkItemCreatedEvent{
		WorkItemID: "wi-1",
		CreatedAt:  now,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.upsertCalls) != 1 {
		t.Fatalf("expected 1 upsert call, got %d", len(store.upsertCalls))
	}
	if store.upsertCalls[0].WorkItemID != "wi-1" {
		t.Errorf("expected work item ID 'wi-1', got %q", store.upsertCalls[0].WorkItemID)
	}
	if store.upsertCalls[0].Status != string(workitemfacade.StatusNew) {
		t.Errorf("expected status %q, got %q", workitemfacade.StatusNew, store.upsertCalls[0].Status)
	}
}

func TestProjection_InboundMessage(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	store := &fakeStore{}
	parties := &fakePartyLookup{data: map[string]partyfacade.PartyInfoDTO{
		"party-anna": {ID: "party-anna", ActorKind: partyfacade.ActorKindHuman, Name: "Anna"},
	}}
	subjects := &fakeSubjectLookup{data: map[string]subjectfacade.SubjectInfoDTO{}}

	subscriber.RegisterProjectionSubscribers(bus, store, parties, subjects)

	err := eventbus.Publish(context.Background(), bus, workitemfacade.InboundMessageRecordedEvent{
		WorkItemID: "wi-1",
		SenderID:   "party-anna",
		Body:       "Hello",
		RecordedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.appendTimelineCalls) != 1 {
		t.Fatalf("expected 1 append timeline call, got %d", len(store.appendTimelineCalls))
	}
	call := store.appendTimelineCalls[0]
	if call.Entry.ActorName != "Anna" {
		t.Errorf("expected actor name 'Anna', got %q", call.Entry.ActorName)
	}
	if call.Entry.EventType != "inbound_message" {
		t.Errorf("expected event type 'inbound_message', got %q", call.Entry.EventType)
	}
}

func TestProjection_StatusChanged(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	store := &fakeStore{}
	parties := &fakePartyLookup{data: map[string]partyfacade.PartyInfoDTO{}}
	subjects := &fakeSubjectLookup{data: map[string]subjectfacade.SubjectInfoDTO{}}

	subscriber.RegisterProjectionSubscribers(bus, store, parties, subjects)

	err := eventbus.Publish(context.Background(), bus, workitemfacade.WorkItemStatusChangedEvent{
		WorkItemID: "wi-1",
		OldStatus:  workitemfacade.StatusNew,
		NewStatus:  workitemfacade.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.updateStatusCalls) != 1 {
		t.Fatalf("expected 1 update status call, got %d", len(store.updateStatusCalls))
	}
	if store.updateStatusCalls[0].Status != string(workitemfacade.StatusInProgress) {
		t.Errorf("expected status %q, got %q", workitemfacade.StatusInProgress, store.updateStatusCalls[0].Status)
	}
}
