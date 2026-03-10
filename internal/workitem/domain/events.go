package domain

import "time"

// Event type constants for the workitem aggregate.
const (
	EventWorkItemCreated              = "workitem.WorkItemCreated.v1"
	EventPartyLinked                  = "workitem.PartyLinkedToWorkItem.v1"
	EventSubjectLinked                = "workitem.SubjectLinkedToWorkItem.v1"
	EventInboundMessageRecorded       = "workitem.InboundMessageRecorded.v1"
	EventAssistantActionRecorded      = "workitem.AssistantActionRecorded.v1"
	EventOutboundMessageRecorded      = "workitem.OutboundMessageRecorded.v1"
	EventWorkItemStatusChanged        = "workitem.WorkItemStatusChanged.v1"
	EventNoteAddedToTimelineEntry     = "workitem.NoteAddedToTimelineEntry.v1"
	EventNoteEditedOnTimelineEntry    = "workitem.NoteEditedOnTimelineEntry.v1"
	EventNoteDeletedFromTimelineEntry = "workitem.NoteDeletedFromTimelineEntry.v1"
)

// DomainEvent is what command methods return. Domain-local, zero platform imports.
type DomainEvent struct {
	EventType string
	Payload   any
}

// WorkItemCreated is emitted when a new work item is created.
type WorkItemCreated struct {
	WorkItemID string    `json:"work_item_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// PartyLinkedToWorkItem is emitted when a party is linked to a work item.
type PartyLinkedToWorkItem struct {
	WorkItemID string    `json:"work_item_id"`
	PartyID    string    `json:"party_id"`
	Role       PartyRole `json:"role"`
}

// SubjectLinkedToWorkItem is emitted when a subject is linked to a work item.
type SubjectLinkedToWorkItem struct {
	WorkItemID string `json:"work_item_id"`
	SubjectID  string `json:"subject_id"`
}

// InboundMessageRecorded is emitted when an inbound message is recorded.
type InboundMessageRecorded struct {
	WorkItemID string    `json:"work_item_id"`
	SenderID   string    `json:"sender_id"`
	Body       string    `json:"body"`
	RecordedAt time.Time `json:"recorded_at"`
}

// AssistantActionRecorded is emitted when an AI assistant performs an action.
type AssistantActionRecorded struct {
	WorkItemID  string      `json:"work_item_id"`
	ActorID     string      `json:"actor_id"`
	ActionKind  ActionKind  `json:"action_kind"`
	Output      string      `json:"output"`
	DraftStatus DraftStatus `json:"draft_status"`
	RecordedAt  time.Time   `json:"recorded_at"`
}

// OutboundMessageRecorded is emitted when an outbound message is confirmed and sent.
type OutboundMessageRecorded struct {
	WorkItemID  string    `json:"work_item_id"`
	ConfirmedBy string    `json:"confirmed_by"`
	Body        string    `json:"body"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// WorkItemStatusChanged is emitted when the work item status changes.
type WorkItemStatusChanged struct {
	WorkItemID string `json:"work_item_id"`
	OldStatus  Status `json:"old_status"`
	NewStatus  Status `json:"new_status"`
}

// NoteAddedToTimelineEntry is emitted when a note is added to a timeline entry.
type NoteAddedToTimelineEntry struct {
	WorkItemID string    `json:"work_item_id"`
	NoteID     string    `json:"note_id"`
	EntryIndex int       `json:"entry_index"`
	AuthorID   string    `json:"author_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// NoteEditedOnTimelineEntry is emitted when a note's body is edited.
type NoteEditedOnTimelineEntry struct {
	WorkItemID string    `json:"work_item_id"`
	NoteID     string    `json:"note_id"`
	Body       string    `json:"body"`
	EditedAt   time.Time `json:"edited_at"`
}

// NoteDeletedFromTimelineEntry is emitted when a note is soft-deleted.
type NoteDeletedFromTimelineEntry struct {
	WorkItemID string    `json:"work_item_id"`
	NoteID     string    `json:"note_id"`
	DeletedAt  time.Time `json:"deleted_at"`
}
