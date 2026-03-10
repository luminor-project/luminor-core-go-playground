package facade

import "time"

// WorkItemCreatedEvent is published when a new work item is created.
type WorkItemCreatedEvent struct {
	WorkItemID string
	CreatedAt  time.Time
}

// PartyLinkedEvent is published when a party is linked to a work item.
type PartyLinkedEvent struct {
	WorkItemID string
	PartyID    string
	Role       PartyRole
}

// SubjectLinkedEvent is published when a subject is linked to a work item.
type SubjectLinkedEvent struct {
	WorkItemID string
	SubjectID  string
}

// InboundMessageRecordedEvent is published when an inbound message is recorded.
type InboundMessageRecordedEvent struct {
	WorkItemID string
	SenderID   string
	Body       string
	RecordedAt time.Time
}

// AssistantActionRecordedEvent is published when an AI assistant performs an action.
type AssistantActionRecordedEvent struct {
	WorkItemID  string
	ActorID     string
	ActionKind  ActionKind
	Output      string
	DraftStatus DraftStatus
	RecordedAt  time.Time
}

// OutboundMessageRecordedEvent is published when an outbound message is confirmed.
type OutboundMessageRecordedEvent struct {
	WorkItemID  string
	ConfirmedBy string
	Body        string
	RecordedAt  time.Time
}

// WorkItemStatusChangedEvent is published when the work item status changes.
type WorkItemStatusChangedEvent struct {
	WorkItemID string
	OldStatus  Status
	NewStatus  Status
}

// NoteAddedToTimelineEntryEvent is published when a note is added to a timeline entry.
type NoteAddedToTimelineEntryEvent struct {
	WorkItemID string
	NoteID     string
	EntryIndex int
	AuthorID   string
	Body       string
	CreatedAt  time.Time
}

// NoteEditedOnTimelineEntryEvent is published when a note is edited.
type NoteEditedOnTimelineEntryEvent struct {
	WorkItemID string
	NoteID     string
	Body       string
	EditedAt   time.Time
}

// NoteDeletedFromTimelineEntryEvent is published when a note is soft-deleted.
type NoteDeletedFromTimelineEntryEvent struct {
	WorkItemID string
	NoteID     string
	DeletedAt  time.Time
}
