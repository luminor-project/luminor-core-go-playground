package domain

import (
	"encoding/json"
	"fmt"
)

var eventFactory = map[string]func() any{
	EventRentalEstablished: func() any { return &RentalEstablished{} },
}

func DeserializeEvent(eventType string, raw json.RawMessage) (any, error) {
	factory, ok := eventFactory[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
	target := factory()
	if err := json.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", eventType, err)
	}
	switch v := target.(type) {
	case *RentalEstablished:
		return *v, nil
	default:
		return nil, fmt.Errorf("unhandled event type: %s", eventType)
	}
}
