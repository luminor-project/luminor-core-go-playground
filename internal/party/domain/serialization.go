package domain

import (
	"encoding/json"
	"fmt"
)

// eventFactory maps event types to factory functions that return a pointer to the target struct.
var eventFactory = map[string]func() any{
	EventPartyRegistered: func() any { return &PartyRegistered{} },
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
	case *PartyRegistered:
		return *v, nil
	default:
		return nil, fmt.Errorf("unhandled event type: %s", eventType)
	}
}
