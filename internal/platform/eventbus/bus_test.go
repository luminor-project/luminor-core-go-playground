package eventbus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventbus"
)

type testEvent struct {
	Message string
}

type otherEvent struct {
	Value int
}

func TestPublish_CallsRegisteredHandlers(t *testing.T) {
	bus := eventbus.New()
	var received []string

	eventbus.Subscribe(bus, func(_ context.Context, e testEvent) error {
		received = append(received, e.Message)
		return nil
	})

	eventbus.Subscribe(bus, func(_ context.Context, e testEvent) error {
		received = append(received, e.Message+"-2")
		return nil
	})

	err := eventbus.Publish(context.Background(), bus, testEvent{Message: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received))
	}
	if received[0] != "hello" {
		t.Errorf("expected 'hello', got %q", received[0])
	}
	if received[1] != "hello-2" {
		t.Errorf("expected 'hello-2', got %q", received[1])
	}
}

func TestPublish_NoHandlers(t *testing.T) {
	bus := eventbus.New()

	err := eventbus.Publish(context.Background(), bus, testEvent{Message: "ignored"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublish_DifferentEventTypes(t *testing.T) {
	bus := eventbus.New()
	var testReceived bool
	var otherReceived bool

	eventbus.Subscribe(bus, func(_ context.Context, _ testEvent) error {
		testReceived = true
		return nil
	})

	eventbus.Subscribe(bus, func(_ context.Context, _ otherEvent) error {
		otherReceived = true
		return nil
	})

	err := eventbus.Publish(context.Background(), bus, testEvent{Message: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !testReceived {
		t.Error("expected testEvent handler to be called")
	}
	if otherReceived {
		t.Error("expected otherEvent handler NOT to be called")
	}
}

func TestPublish_HandlerError_StopsExecution(t *testing.T) {
	bus := eventbus.New()
	expectedErr := errors.New("handler failed")

	eventbus.Subscribe(bus, func(_ context.Context, _ testEvent) error {
		return expectedErr
	})

	eventbus.Subscribe(bus, func(_ context.Context, _ testEvent) error {
		t.Error("second handler should not be called after first error")
		return nil
	})

	err := eventbus.Publish(context.Background(), bus, testEvent{Message: "fail"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}
