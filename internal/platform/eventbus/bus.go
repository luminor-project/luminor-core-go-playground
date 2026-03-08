package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

type Bus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]any
}

func New() *Bus {
	return &Bus{
		handlers: make(map[reflect.Type][]any),
	}
}

// Subscribe registers a handler for a specific event type.
// The handler must be a func(context.Context, T) error where T is the event type.
func Subscribe[T any](bus *Bus, handler func(context.Context, T) error) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	var zero T
	t := reflect.TypeOf(zero)
	bus.handlers[t] = append(bus.handlers[t], handler)

	slog.Debug("event subscriber registered", "event", t.Name())
}

// Publish dispatches an event to all registered handlers synchronously.
func Publish[T any](ctx context.Context, bus *Bus, event T) error {
	bus.mu.RLock()
	t := reflect.TypeOf(event)
	handlers := bus.handlers[t]
	bus.mu.RUnlock()

	slog.Debug("publishing event", "event", t.Name(), "handlers", len(handlers))

	for _, h := range handlers {
		// SAFETY: The type assertion cannot fail at runtime. Subscribe[T] stores
		// handlers keyed by reflect.TypeOf(T), and Publish[T] looks them up by
		// the same key. Because both functions share the same generic type
		// parameter T, the stored handler is always func(context.Context, T) error.
		handler, ok := h.(func(context.Context, T) error)
		if !ok {
			return fmt.Errorf("invalid handler type for event %s", t.Name())
		}
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handler error for event %s: %w", t.Name(), err)
		}
	}

	return nil
}
