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
	Role       string
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
	ActionKind  string
	Output      string
	DraftStatus string
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
	OldStatus  string
	NewStatus  string
}
