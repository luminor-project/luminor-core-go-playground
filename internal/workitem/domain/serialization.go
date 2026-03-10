package domain

import (
	"encoding/json"
	"fmt"
)

// eventFactory maps event types to factory functions that return a pointer to the target struct.
var eventFactory = map[string]func() any{
	EventWorkItemCreated:              func() any { return &WorkItemCreated{} },
	EventPartyLinked:                  func() any { return &PartyLinkedToWorkItem{} },
	EventSubjectLinked:                func() any { return &SubjectLinkedToWorkItem{} },
	EventInboundMessageRecorded:       func() any { return &InboundMessageRecorded{} },
	EventAssistantActionRecorded:      func() any { return &AssistantActionRecorded{} },
	EventOutboundMessageRecorded:      func() any { return &OutboundMessageRecorded{} },
	EventWorkItemStatusChanged:        func() any { return &WorkItemStatusChanged{} },
	EventNoteAddedToTimelineEntry:     func() any { return &NoteAddedToTimelineEntry{} },
	EventNoteEditedOnTimelineEntry:    func() any { return &NoteEditedOnTimelineEntry{} },
	EventNoteDeletedFromTimelineEntry: func() any { return &NoteDeletedFromTimelineEntry{} },
}

// DeserializeEvent converts a raw JSON payload into a typed event struct based on the event type.
func DeserializeEvent(eventType string, raw json.RawMessage) (any, error) {
	factory, ok := eventFactory[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	target := factory()
	if err := json.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", eventType, err)
	}

	// Dereference pointer to return value type, matching Apply() expectations.
	switch v := target.(type) {
	case *WorkItemCreated:
		return *v, nil
	case *PartyLinkedToWorkItem:
		return *v, nil
	case *SubjectLinkedToWorkItem:
		return *v, nil
	case *InboundMessageRecorded:
		return *v, nil
	case *AssistantActionRecorded:
		return *v, nil
	case *OutboundMessageRecorded:
		return *v, nil
	case *WorkItemStatusChanged:
		return *v, nil
	case *NoteAddedToTimelineEntry:
		return *v, nil
	case *NoteEditedOnTimelineEntry:
		return *v, nil
	case *NoteDeletedFromTimelineEntry:
		return *v, nil
	default:
		return nil, fmt.Errorf("unhandled event type: %s", eventType)
	}
}
