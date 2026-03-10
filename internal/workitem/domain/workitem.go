package domain

import (
	"errors"
	"time"
)

// WorkItem status constants.
const (
	StatusNew                 = "new"
	StatusInProgress          = "in_progress"
	StatusPendingConfirmation = "pending_confirmation"
	StatusResolved            = "resolved"
)

var (
	ErrAlreadyCreated    = errors.New("work item already created")
	ErrNotCreated        = errors.New("work item not yet created")
	ErrNoPendingDraft    = errors.New("no pending draft to confirm")
	ErrAlreadyConfirmed  = errors.New("outbound message already confirmed")
	ErrInvalidActionKind = errors.New("invalid action kind")
	ErrAlreadyResolved   = errors.New("work item already resolved")
)

// WorkItem is the event-sourced aggregate for case work items.
type WorkItem struct {
	ID              string
	Status          string
	Version         int
	PartyIDs        []string
	SubjectID       string
	Created         bool
	HasPendingDraft bool
	Confirmed       bool
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
		// No state change beyond what's already tracked

	case EventAssistantActionRecorded:
		e := payload.(AssistantActionRecorded)
		if e.DraftStatus == "pending" {
			w.HasPendingDraft = true
		}

	case EventOutboundMessageRecorded:
		w.HasPendingDraft = false
		w.Confirmed = true

	case EventWorkItemStatusChanged:
		e := payload.(WorkItemStatusChanged)
		w.Status = e.NewStatus
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
			Role:       "sender",
		}},
		{EventType: EventPartyLinked, Payload: PartyLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			PartyID:    cmd.HandlerPartyID,
			Role:       "handler",
		}},
		{EventType: EventPartyLinked, Payload: PartyLinkedToWorkItem{
			WorkItemID: cmd.WorkItemID,
			PartyID:    cmd.AgentPartyID,
			Role:       "agent",
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
	ActionKind  string // "lookup", "draft"
	Output      string
	DraftStatus string // "" for lookup, "pending" for draft
}

// RecordAssistantAction records an AI assistant action on the work item.
func (w *WorkItem) RecordAssistantAction(cmd AssistantActionCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}
	if w.Status == StatusResolved {
		return nil, ErrAlreadyResolved
	}
	if cmd.ActionKind != "lookup" && cmd.ActionKind != "draft" {
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

	if cmd.DraftStatus == "pending" {
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

// SetStatusCmd holds the data for changing the work item status.
type SetStatusCmd struct {
	WorkItemID string
	NewStatus  string
}

// SetStatus changes the work item status.
func (w *WorkItem) SetStatus(cmd SetStatusCmd) ([]DomainEvent, error) {
	if !w.Created {
		return nil, ErrNotCreated
	}

	events := []DomainEvent{
		{EventType: EventWorkItemStatusChanged, Payload: WorkItemStatusChanged{
			WorkItemID: cmd.WorkItemID,
			OldStatus:  w.Status,
			NewStatus:  cmd.NewStatus,
		}},
	}

	return events, nil
}
