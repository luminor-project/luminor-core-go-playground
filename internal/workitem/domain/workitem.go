package domain

import (
	"errors"
	"time"
)

var (
	ErrAlreadyCreated     = errors.New("work item already created")
	ErrNotCreated         = errors.New("work item not yet created")
	ErrNoPendingDraft     = errors.New("no pending draft to confirm")
	ErrAlreadyConfirmed   = errors.New("outbound message already confirmed")
	ErrInvalidActionKind  = errors.New("invalid action kind")
	ErrAlreadyResolved    = errors.New("work item already resolved")
	ErrNoteNotFound       = errors.New("note not found")
	ErrNoteAlreadyDeleted = errors.New("note already deleted")
	ErrInvalidEntryIndex  = errors.New("invalid timeline entry index")
)

// WorkItem is the event-sourced aggregate for case work items.
type WorkItem struct {
	ID                 string
	Status             Status
	Version            int
	PartyIDs           []string
	SubjectID          string
	Created            bool
	HasPendingDraft    bool
	Confirmed          bool
	TimelineEntryCount int
	NoteIDs            map[string]bool // noteID → deleted?
}

// Apply reconstitutes state from a single event payload.
func (w *WorkItem) Apply(eventType string, payload any) {
	switch eventType {
	case EventWorkItemCreated:
		e := payload.(WorkItemCreated)
		w.ID = e.WorkItemID
		w.Status = StatusNew
		w.Created = true

	case EventPartyLinked:
		e := payload.(PartyLinkedToWorkItem)
		w.PartyIDs = append(w.PartyIDs, e.PartyID)

	case EventSubjectLinked:
		e := payload.(SubjectLinkedToWorkItem)
		w.SubjectID = e.SubjectID

	case EventInboundMessageRecorded:
		w.TimelineEntryCount++

	case EventAssistantActionRecorded:
		e := payload.(AssistantActionRecorded)
		if e.DraftStatus == DraftStatusPending {
			w.HasPendingDraft = true
		}
		w.TimelineEntryCount++

	case EventOutboundMessageRecorded:
		w.HasPendingDraft = false
		w.Confirmed = true
		w.TimelineEntryCount++

	case EventWorkItemStatusChanged:
		e := payload.(WorkItemStatusChanged)
		w.Status = e.NewStatus

	case EventNoteAddedToTimelineEntry:
		e := payload.(NoteAddedToTimelineEntry)
		if w.NoteIDs == nil {
			w.NoteIDs = make(map[string]bool)
		}
		w.NoteIDs[e.NoteID] = false

	case EventNoteEditedOnTimelineEntry:
		// No state change beyond what's tracked

	case EventNoteDeletedFromTimelineEntry:
		e := payload.(NoteDeletedFromTimelineEntry)
		w.NoteIDs[e.NoteID] = true
	}

	w.Version++
}

// IntakeCmd holds the data needed to create a work item and record an inbound message.
type IntakeCmd struct {
	WorkItemID     string
	SenderPartyID  string
	SubjectID      string
	Body           string
	HandlerPartyID string
	AgentPartyID   string
}

// IntakeInboundMessage creates a new work item and records the initial inbound message.
func (w *WorkItem) IntakeInboundMessage(cmd IntakeCmd) ([]DomainEvent, error) {
	if w.Created {
		return nil, ErrAlreadyCreated
	}

	now := time.Now()

	events := []DomainEvent{
		{EventType: EventWorkItemCreated, Payload: WorkItemCreated{
			WorkItemID: cmd.WorkItemID,
			CreatedAt:  now,
		}},
		{EventType: EventPartyLinked, Payload: PartyLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			PartyID:    cmd.SenderPartyID,
			Role:       PartyRoleSender,
		}},
		{EventType: EventPartyLinked, Payload: PartyLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			PartyID:    cmd.HandlerPartyID,
			Role:       PartyRoleHandler,
		}},
		{EventType: EventPartyLinked, Payload: PartyLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			PartyID:    cmd.AgentPartyID,
			Role:       PartyRoleAgent,
		}},
		{EventType: EventSubjectLinked, Payload: SubjectLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			SubjectID:  cmd.SubjectID,
		}},
		{EventType: EventInboundMessageRecorded, Payload: InboundMessageRecorded{
			WorkItemID: cmd.WorkItemID,
			SenderID:   cmd.SenderPartyID,
			Body:       cmd.Body,
			RecordedAt: now,
		}},
		{EventType: EventWorkItemStatusChanged, Payload: WorkItemStatusChanged{
			WorkItemID: cmd.WorkItemID,
			OldStatus:  "",
			NewStatus:  StatusInProgress,
		}},
	}

	return events, nil
}

// AssistantActionCmd holds the data for recording an AI assistant action.
type AssistantActionCmd struct {
	WorkItemID  string
	ActorID     string
	ActionKind  ActionKind
	Output      string
	DraftStatus DraftStatus
}

// RecordAssistantAction records an AI assistant action on the work item.
func (w *WorkItem) RecordAssistantAction(cmd AssistantActionCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	if w.Status == StatusResolved {
		return nil, ErrAlreadyResolved
	}
	if cmd.ActionKind != ActionKindLookup && cmd.ActionKind != ActionKindDraft {
		return nil, ErrInvalidActionKind
	}

	now := time.Now()

	events := []DomainEvent{
		{EventType: EventAssistantActionRecorded, Payload: AssistantActionRecorded{
			WorkItemID:  cmd.WorkItemID,
			ActorID:     cmd.ActorID,
			ActionKind:  cmd.ActionKind,
			Output:      cmd.Output,
			DraftStatus: cmd.DraftStatus,
			RecordedAt:  now,
		}},
	}

	if cmd.DraftStatus == DraftStatusPending {
		events = append(events, DomainEvent{
			EventType: EventWorkItemStatusChanged,
			Payload: WorkItemStatusChanged{
				WorkItemID: cmd.WorkItemID,
				OldStatus:  w.Status,
				NewStatus:  StatusPendingConfirmation,
			},
		})
	}

	return events, nil
}

// ConfirmCmd holds the data for confirming an outbound message.
type ConfirmCmd struct {
	WorkItemID  string
	ConfirmedBy string
	Body        string
}

// ConfirmOutboundMessage confirms and records the outbound message.
func (w *WorkItem) ConfirmOutboundMessage(cmd ConfirmCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	if w.Confirmed {
		return nil, ErrAlreadyConfirmed
	}
	if !w.HasPendingDraft {
		return nil, ErrNoPendingDraft
	}

	now := time.Now()

	events := []DomainEvent{
		{EventType: EventOutboundMessageRecorded, Payload: OutboundMessageRecorded{
			WorkItemID:  cmd.WorkItemID,
			ConfirmedBy: cmd.ConfirmedBy,
			Body:        cmd.Body,
			RecordedAt:  now,
		}},
		{EventType: EventWorkItemStatusChanged, Payload: WorkItemStatusChanged{
			WorkItemID: cmd.WorkItemID,
			OldStatus:  w.Status,
			NewStatus:  StatusResolved,
		}},
	}

	return events, nil
}

// AddNoteCmd holds the data for adding a note to a timeline entry.
type AddNoteCmd struct {
	WorkItemID string
	NoteID     string
	EntryIndex int
	AuthorID   string
	Body       string
}

// AddNote adds a note to a timeline entry.
func (w *WorkItem) AddNote(cmd AddNoteCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	if cmd.EntryIndex < 0 || cmd.EntryIndex >= w.TimelineEntryCount {
		return nil, ErrInvalidEntryIndex
	}

	return []DomainEvent{
		{EventType: EventNoteAddedToTimelineEntry, Payload: NoteAddedToTimelineEntry{
			WorkItemID: cmd.WorkItemID,
			NoteID:     cmd.NoteID,
			EntryIndex: cmd.EntryIndex,
			AuthorID:   cmd.AuthorID,
			Body:       cmd.Body,
			CreatedAt:  time.Now(),
		}},
	}, nil
}

// EditNoteCmd holds the data for editing a note's body.
type EditNoteCmd struct {
	WorkItemID string
	NoteID     string
	Body       string
}

// EditNote edits an existing note's body.
func (w *WorkItem) EditNote(cmd EditNoteCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	deleted, exists := w.NoteIDs[cmd.NoteID]
	if !exists {
		return nil, ErrNoteNotFound
	}
	if deleted {
		return nil, ErrNoteAlreadyDeleted
	}

	return []DomainEvent{
		{EventType: EventNoteEditedOnTimelineEntry, Payload: NoteEditedOnTimelineEntry{
			WorkItemID: cmd.WorkItemID,
			NoteID:     cmd.NoteID,
			Body:       cmd.Body,
			EditedAt:   time.Now(),
		}},
	}, nil
}

// DeleteNoteCmd holds the data for soft-deleting a note.
type DeleteNoteCmd struct {
	WorkItemID string
	NoteID     string
}

// DeleteNote soft-deletes a note.
func (w *WorkItem) DeleteNote(cmd DeleteNoteCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	deleted, exists := w.NoteIDs[cmd.NoteID]
	if !exists {
		return nil, ErrNoteNotFound
	}
	if deleted {
		return nil, ErrNoteAlreadyDeleted
	}

	return []DomainEvent{
		{EventType: EventNoteDeletedFromTimelineEntry, Payload: NoteDeletedFromTimelineEntry{
			WorkItemID: cmd.WorkItemID,
			NoteID:     cmd.NoteID,
			DeletedAt:  time.Now(),
		}},
	}, nil
}
