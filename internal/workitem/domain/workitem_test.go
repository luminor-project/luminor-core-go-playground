package domain_test

import (
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/workitem/domain"
)

func applyAll(w *domain.WorkItem, events []domain.DomainEvent) {
	for _, e := range events {
		w.Apply(e.EventType, e.Payload)
	}
}

func TestIntakeInboundMessage_ProducesExpectedEvents(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	events, err := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "Ich möchte meinen Mietvertrag verlängern.",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: Created + 3 PartyLinked + SubjectLinked + InboundMessage + StatusChanged
	if len(events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(events))
	}
	if events[0].EventType != domain.EventWorkItemCreated {
		t.Errorf("event 0: expected %s, got %s", domain.EventWorkItemCreated, events[0].EventType)
	}
	if events[1].EventType != domain.EventPartyLinked {
		t.Errorf("event 1: expected %s, got %s", domain.EventPartyLinked, events[1].EventType)
	}
	if events[5].EventType != domain.EventInboundMessageRecorded {
		t.Errorf("event 5: expected %s, got %s", domain.EventInboundMessageRecorded, events[5].EventType)
	}
	if events[6].EventType != domain.EventWorkItemStatusChanged {
		t.Errorf("event 6: expected %s, got %s", domain.EventWorkItemStatusChanged, events[6].EventType)
	}
}

func TestIntakeInboundMessage_AlreadyCreated(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	events, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, events)

	_, err := w.IntakeInboundMessage(domain.IntakeCmd{WorkItemID: "wi-1"})
	if err != domain.ErrAlreadyCreated {
		t.Fatalf("expected ErrAlreadyCreated, got: %v", err)
	}
}

func TestRecordAssistantAction_Lookup(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	intakeEvents, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, intakeEvents)

	events, err := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID:  "wi-1",
		ActorID:     "party-ki-assistent",
		ActionKind:  domain.ActionKindLookup,
		Output:      "contract data",
		DraftStatus: domain.DraftStatusNone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventAssistantActionRecorded {
		t.Errorf("expected %s, got %s", domain.EventAssistantActionRecorded, events[0].EventType)
	}
}

func TestRecordAssistantAction_Draft_ProducesStatusChange(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	intakeEvents, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, intakeEvents)

	lookupEvents, _ := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1",
		ActorID:    "party-ki-assistent",
		ActionKind: domain.ActionKindLookup,
		Output:     "contract data",
	})
	applyAll(w, lookupEvents)

	events, err := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID:  "wi-1",
		ActorID:     "party-ki-assistent",
		ActionKind:  domain.ActionKindDraft,
		Output:      "draft response",
		DraftStatus: domain.DraftStatusPending,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (action + status change), got %d", len(events))
	}
	if events[1].EventType != domain.EventWorkItemStatusChanged {
		t.Errorf("expected %s, got %s", domain.EventWorkItemStatusChanged, events[1].EventType)
	}
	statusPayload := events[1].Payload.(domain.WorkItemStatusChanged)
	if statusPayload.NewStatus != domain.StatusPendingConfirmation {
		t.Errorf("expected status %s, got %s", domain.StatusPendingConfirmation, statusPayload.NewStatus)
	}
}

func TestConfirmOutboundMessage(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	// Intake
	intakeEvents, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, intakeEvents)

	// Lookup
	lookupEvents, _ := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1",
		ActorID:    "party-ki-assistent",
		ActionKind: domain.ActionKindLookup,
		Output:     "contract data",
	})
	applyAll(w, lookupEvents)

	// Draft
	draftEvents, _ := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID:  "wi-1",
		ActorID:     "party-ki-assistent",
		ActionKind:  domain.ActionKindDraft,
		Output:      "draft response",
		DraftStatus: domain.DraftStatusPending,
	})
	applyAll(w, draftEvents)

	// Confirm
	events, err := w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID:  "wi-1",
		ConfirmedBy: "party-sarah",
		Body:        "confirmed response",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (outbound + status), got %d", len(events))
	}
	if events[0].EventType != domain.EventOutboundMessageRecorded {
		t.Errorf("expected %s, got %s", domain.EventOutboundMessageRecorded, events[0].EventType)
	}
	statusPayload := events[1].Payload.(domain.WorkItemStatusChanged)
	if statusPayload.NewStatus != domain.StatusResolved {
		t.Errorf("expected status %s, got %s", domain.StatusResolved, statusPayload.NewStatus)
	}
}

func TestConfirmOutboundMessage_NoPendingDraft(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	intakeEvents, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, intakeEvents)

	_, err := w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID:  "wi-1",
		ConfirmedBy: "party-sarah",
		Body:        "response",
	})
	if err != domain.ErrNoPendingDraft {
		t.Fatalf("expected ErrNoPendingDraft, got: %v", err)
	}
}

func TestConfirmOutboundMessage_AlreadyConfirmed(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	// Full golden path up to confirmation
	intakeEvents, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, intakeEvents)

	lookupEvents, _ := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent", ActionKind: domain.ActionKindLookup, Output: "data",
	})
	applyAll(w, lookupEvents)

	draftEvents, _ := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent", ActionKind: domain.ActionKindDraft, Output: "draft", DraftStatus: domain.DraftStatusPending,
	})
	applyAll(w, draftEvents)

	confirmEvents, _ := w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID: "wi-1", ConfirmedBy: "party-sarah", Body: "confirmed",
	})
	applyAll(w, confirmEvents)

	// Second confirm should fail
	_, err := w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID: "wi-1", ConfirmedBy: "party-sarah", Body: "again",
	})
	if err != domain.ErrAlreadyConfirmed {
		t.Fatalf("expected ErrAlreadyConfirmed, got: %v", err)
	}
}

func TestGoldenPath_FinalState(t *testing.T) {
	t.Parallel()
	w := &domain.WorkItem{}

	// Intake
	events, _ := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "Ich möchte meinen Mietvertrag verlängern.",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	applyAll(w, events)

	if w.Status != domain.StatusInProgress {
		t.Errorf("after intake: expected status %s, got %s", domain.StatusInProgress, w.Status)
	}

	// Lookup
	events, _ = w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent", ActionKind: domain.ActionKindLookup, Output: "contract data",
	})
	applyAll(w, events)

	// Draft
	events, _ = w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent", ActionKind: domain.ActionKindDraft, Output: "draft", DraftStatus: domain.DraftStatusPending,
	})
	applyAll(w, events)

	if w.Status != domain.StatusPendingConfirmation {
		t.Errorf("after draft: expected status %s, got %s", domain.StatusPendingConfirmation, w.Status)
	}
	if !w.HasPendingDraft {
		t.Error("after draft: expected HasPendingDraft=true")
	}

	// Confirm
	events, _ = w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID: "wi-1", ConfirmedBy: "party-sarah", Body: "confirmed response",
	})
	applyAll(w, events)

	if w.Status != domain.StatusResolved {
		t.Errorf("after confirm: expected status %s, got %s", domain.StatusResolved, w.Status)
	}
	if !w.Confirmed {
		t.Error("after confirm: expected Confirmed=true")
	}
	if w.HasPendingDraft {
		t.Error("after confirm: expected HasPendingDraft=false")
	}
	if len(w.PartyIDs) != 3 {
		t.Errorf("expected 3 party IDs, got %d", len(w.PartyIDs))
	}
	if w.SubjectID != "subject-flussufer-12a" {
		t.Errorf("expected subject ID 'subject-flussufer-12a', got %q", w.SubjectID)
	}
}
