package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// StoredEvent represents a persisted event in the event store.
type StoredEvent struct {
	ID            string
	StreamID      string
	StreamVersion int
	EventType     string
	Payload       json.RawMessage
	CausationID   string
	CorrelationID string
	RecordedAt    time.Time
}

// UncommittedEvent represents an event that has not yet been persisted.
type UncommittedEvent struct {
	EventType     string
	Payload       any
	CausationID   string
	CorrelationID string
}

// ErrConcurrencyConflict is returned when an append fails due to a stream version mismatch.
var ErrConcurrencyConflict = errors.New("concurrency conflict: stream version mismatch")

// Store is the event store interface.
type Store interface {
	Append(ctx context.Context, streamID string, expectedVersion int, events []UncommittedEvent) ([]StoredEvent, error)
	LoadStream(ctx context.Context, streamID string) ([]StoredEvent, error)
}
