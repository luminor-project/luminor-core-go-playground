package domain_test

import (
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/clock"
	"github.com/luminor-project/luminor-core-go-playground/internal/workitem/domain"
)

var fixedClock = clock.NewFixed(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))

// stepClock returns a new time on each call, advancing by step.
type stepClock struct {
	current time.Time
	step    time.Duration
}

func (c *stepClock) Now() time.Time {
	t := c.current
	c.current = c.current.Add(c.step)
	return t
}

func applyAll(w *domain.WorkItem, events []domain.DomainEvent) {
	for _, e := range events {
		w.Apply(e.EventType, e.Payload)
	}
}

func TestIntakeInboundMessage_ProducesExpectedEvents(t *testing.T) {
	t.Parallel()
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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
	w := domain.NewWorkItem(fixedClock)

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

// workItemWithIntake returns a WorkItem after a successful intake (Created=true, TimelineEntryCount=1).
func workItemWithIntake(t *testing.T) *domain.WorkItem {
	t.Helper()
	w := domain.NewWorkItem(fixedClock)
	events, err := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "test",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	if err != nil {
		t.Fatalf("intake failed: %v", err)
	}
	applyAll(w, events)
	return w
}

func TestAddNote(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	events, err := w.AddNote(domain.AddNoteCmd{
		WorkItemID: "wi-1",
		NoteID:     "note-1",
		EntryIndex: 0,
		AuthorID:   "party-sarah",
		Body:       "Internal remark",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventNoteAddedToTimelineEntry {
		t.Errorf("expected %s, got %s", domain.EventNoteAddedToTimelineEntry, events[0].EventType)
	}

	applyAll(w, events)
	if w.NoteIDs == nil {
		t.Fatal("expected NoteIDs to be initialized")
	}
	if deleted, ok := w.NoteIDs["note-1"]; !ok || deleted {
		t.Error("expected note-1 to exist and not be deleted")
	}
}

func TestAddNote_NotCreated(t *testing.T) {
	t.Parallel()
	w := domain.NewWorkItem(fixedClock)
	_, err := w.AddNote(domain.AddNoteCmd{NoteID: "note-1", EntryIndex: 0})
	if err != domain.ErrNotCreated {
		t.Fatalf("expected ErrNotCreated, got: %v", err)
	}
}

func TestAddNote_InvalidEntryIndex(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	_, err := w.AddNote(domain.AddNoteCmd{NoteID: "note-1", EntryIndex: -1})
	if err != domain.ErrInvalidEntryIndex {
		t.Fatalf("expected ErrInvalidEntryIndex for negative index, got: %v", err)
	}

	_, err = w.AddNote(domain.AddNoteCmd{NoteID: "note-1", EntryIndex: 99})
	if err != domain.ErrInvalidEntryIndex {
		t.Fatalf("expected ErrInvalidEntryIndex for out-of-range index, got: %v", err)
	}
}

func TestEditNote(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	addEvents, _ := w.AddNote(domain.AddNoteCmd{
		WorkItemID: "wi-1", NoteID: "note-1", EntryIndex: 0, AuthorID: "party-sarah", Body: "original",
	})
	applyAll(w, addEvents)

	events, err := w.EditNote(domain.EditNoteCmd{
		WorkItemID: "wi-1", NoteID: "note-1", Body: "updated",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventNoteEditedOnTimelineEntry {
		t.Errorf("expected %s, got %s", domain.EventNoteEditedOnTimelineEntry, events[0].EventType)
	}
}

func TestEditNote_NotFound(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	_, err := w.EditNote(domain.EditNoteCmd{NoteID: "nonexistent", Body: "x"})
	if err != domain.ErrNoteNotFound {
		t.Fatalf("expected ErrNoteNotFound, got: %v", err)
	}
}

func TestEditNote_AlreadyDeleted(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	addEvents, _ := w.AddNote(domain.AddNoteCmd{
		WorkItemID: "wi-1", NoteID: "note-1", EntryIndex: 0, AuthorID: "party-sarah", Body: "original",
	})
	applyAll(w, addEvents)

	delEvents, _ := w.DeleteNote(domain.DeleteNoteCmd{WorkItemID: "wi-1", NoteID: "note-1"})
	applyAll(w, delEvents)

	_, err := w.EditNote(domain.EditNoteCmd{NoteID: "note-1", Body: "updated"})
	if err != domain.ErrNoteAlreadyDeleted {
		t.Fatalf("expected ErrNoteAlreadyDeleted, got: %v", err)
	}
}

func TestDeleteNote(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	addEvents, _ := w.AddNote(domain.AddNoteCmd{
		WorkItemID: "wi-1", NoteID: "note-1", EntryIndex: 0, AuthorID: "party-sarah", Body: "to delete",
	})
	applyAll(w, addEvents)

	events, err := w.DeleteNote(domain.DeleteNoteCmd{WorkItemID: "wi-1", NoteID: "note-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != domain.EventNoteDeletedFromTimelineEntry {
		t.Errorf("expected %s, got %s", domain.EventNoteDeletedFromTimelineEntry, events[0].EventType)
	}

	applyAll(w, events)
	if !w.NoteIDs["note-1"] {
		t.Error("expected note-1 to be marked as deleted")
	}
}

func TestDeleteNote_NotFound(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	_, err := w.DeleteNote(domain.DeleteNoteCmd{NoteID: "nonexistent"})
	if err != domain.ErrNoteNotFound {
		t.Fatalf("expected ErrNoteNotFound, got: %v", err)
	}
}

func TestDeleteNote_AlreadyDeleted(t *testing.T) {
	t.Parallel()
	w := workItemWithIntake(t)

	addEvents, _ := w.AddNote(domain.AddNoteCmd{
		WorkItemID: "wi-1", NoteID: "note-1", EntryIndex: 0, AuthorID: "party-sarah", Body: "x",
	})
	applyAll(w, addEvents)

	delEvents, _ := w.DeleteNote(domain.DeleteNoteCmd{WorkItemID: "wi-1", NoteID: "note-1"})
	applyAll(w, delEvents)

	_, err := w.DeleteNote(domain.DeleteNoteCmd{NoteID: "note-1"})
	if err != domain.ErrNoteAlreadyDeleted {
		t.Fatalf("expected ErrNoteAlreadyDeleted, got: %v", err)
	}
}

func TestGoldenPath_FinalState(t *testing.T) {
	t.Parallel()
	w := domain.NewWorkItem(fixedClock)

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

func TestGoldenPath_EventTimestamps(t *testing.T) {
	t.Parallel()

	// Each command calls clock.Now() exactly once. A stepping clock lets us
	// assert that (a) events within a single command share a timestamp and
	// (b) timestamps across commands are causally ordered.
	t0 := time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC)
	clk := &stepClock{current: t0, step: time.Minute}
	w := domain.NewWorkItem(clk)

	// Step 0 (t0): Intake — creates 7 events, all stamped with t0.
	intakeEvents, err := w.IntakeInboundMessage(domain.IntakeCmd{
		WorkItemID:     "wi-1",
		SenderPartyID:  "party-anna-schmidt",
		SubjectID:      "subject-flussufer-12a",
		Body:           "Anfrage Mietvertrag",
		HandlerPartyID: "party-sarah",
		AgentPartyID:   "party-ki-assistent",
	})
	if err != nil {
		t.Fatalf("intake: %v", err)
	}
	applyAll(w, intakeEvents)

	createdAt := intakeEvents[0].Payload.(domain.WorkItemCreated).CreatedAt
	recordedAt := intakeEvents[5].Payload.(domain.InboundMessageRecorded).RecordedAt
	if createdAt != t0 {
		t.Errorf("WorkItemCreated.CreatedAt = %v, want %v", createdAt, t0)
	}
	if recordedAt != t0 {
		t.Errorf("InboundMessageRecorded.RecordedAt = %v, want %v (same command → same timestamp)", recordedAt, t0)
	}

	// Step 1 (t0+1m): Lookup
	t1 := t0.Add(time.Minute)
	lookupEvents, err := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent",
		ActionKind: domain.ActionKindLookup, Output: "Vertragsdaten",
	})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	applyAll(w, lookupEvents)

	lookupAt := lookupEvents[0].Payload.(domain.AssistantActionRecorded).RecordedAt
	if lookupAt != t1 {
		t.Errorf("lookup RecordedAt = %v, want %v", lookupAt, t1)
	}

	// Step 2 (t0+2m): Draft
	t2 := t0.Add(2 * time.Minute)
	draftEvents, err := w.RecordAssistantAction(domain.AssistantActionCmd{
		WorkItemID: "wi-1", ActorID: "party-ki-assistent",
		ActionKind: domain.ActionKindDraft, Output: "Entwurf",
		DraftStatus: domain.DraftStatusPending,
	})
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	applyAll(w, draftEvents)

	draftAt := draftEvents[0].Payload.(domain.AssistantActionRecorded).RecordedAt
	if draftAt != t2 {
		t.Errorf("draft RecordedAt = %v, want %v", draftAt, t2)
	}

	// Step 3 (t0+3m): Confirm
	t3 := t0.Add(3 * time.Minute)
	confirmEvents, err := w.ConfirmOutboundMessage(domain.ConfirmCmd{
		WorkItemID: "wi-1", ConfirmedBy: "party-sarah", Body: "Bestätigte Antwort",
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	applyAll(w, confirmEvents)

	confirmAt := confirmEvents[0].Payload.(domain.OutboundMessageRecorded).RecordedAt
	if confirmAt != t3 {
		t.Errorf("confirm RecordedAt = %v, want %v", confirmAt, t3)
	}

	// Verify causal ordering across the full timeline.
	if !createdAt.Before(lookupAt) || !lookupAt.Before(draftAt) || !draftAt.Before(confirmAt) {
		t.Errorf("timestamps not causally ordered: created=%v lookup=%v draft=%v confirm=%v",
			createdAt, lookupAt, draftAt, confirmAt)
	}
}
