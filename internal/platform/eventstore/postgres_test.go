package eventstore_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/eventstore"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://luminor:luminor@localhost:5442/luminor?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func cleanupStream(t *testing.T, pool *pgxpool.Pool, streamID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "DELETE FROM events WHERE stream_id = $1", streamID)
	if err != nil {
		t.Fatalf("cleanup stream: %v", err)
	}
}

func TestAppend_NewStream(t *testing.T) {
	pool := testPool(t)
	store := eventstore.NewPostgresStore(pool)
	ctx := context.Background()
	streamID := "test-append-new-" + t.Name()
	t.Cleanup(func() { cleanupStream(t, pool, streamID) })

	events := []eventstore.UncommittedEvent{
		{EventType: "test.Created.v1", Payload: map[string]string{"id": "1"}},
		{EventType: "test.Updated.v1", Payload: map[string]string{"value": "hello"}},
	}

	stored, err := store.Append(ctx, streamID, 0, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored events, got %d", len(stored))
	}
	if stored[0].StreamVersion != 1 {
		t.Errorf("expected version 1, got %d", stored[0].StreamVersion)
	}
	if stored[1].StreamVersion != 2 {
		t.Errorf("expected version 2, got %d", stored[1].StreamVersion)
	}
	if stored[0].ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestAppend_ExistingStream(t *testing.T) {
	pool := testPool(t)
	store := eventstore.NewPostgresStore(pool)
	ctx := context.Background()
	streamID := "test-append-existing-" + t.Name()
	t.Cleanup(func() { cleanupStream(t, pool, streamID) })

	_, err := store.Append(ctx, streamID, 0, []eventstore.UncommittedEvent{
		{EventType: "test.Created.v1", Payload: map[string]string{"id": "1"}},
	})
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	stored, err := store.Append(ctx, streamID, 1, []eventstore.UncommittedEvent{
		{EventType: "test.Updated.v1", Payload: map[string]string{"value": "world"}},
	})
	if err != nil {
		t.Fatalf("second append: %v", err)
	}
	if stored[0].StreamVersion != 2 {
		t.Errorf("expected version 2, got %d", stored[0].StreamVersion)
	}
}

func TestAppend_ConcurrencyConflict(t *testing.T) {
	pool := testPool(t)
	store := eventstore.NewPostgresStore(pool)
	ctx := context.Background()
	streamID := "test-concurrency-" + t.Name()
	t.Cleanup(func() { cleanupStream(t, pool, streamID) })

	_, err := store.Append(ctx, streamID, 0, []eventstore.UncommittedEvent{
		{EventType: "test.Created.v1", Payload: map[string]string{"id": "1"}},
	})
	if err != nil {
		t.Fatalf("first append: %v", err)
	}

	// Append with wrong expected version (0 instead of 1)
	_, err = store.Append(ctx, streamID, 0, []eventstore.UncommittedEvent{
		{EventType: "test.Conflict.v1", Payload: map[string]string{"bad": "true"}},
	})
	if err != eventstore.ErrConcurrencyConflict {
		t.Fatalf("expected ErrConcurrencyConflict, got: %v", err)
	}
}

func TestLoadStream_Empty(t *testing.T) {
	pool := testPool(t)
	store := eventstore.NewPostgresStore(pool)
	ctx := context.Background()

	events, err := store.LoadStream(ctx, "nonexistent-stream")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected empty slice, got %d events", len(events))
	}
}

func TestLoadStream_ReturnsInOrder(t *testing.T) {
	pool := testPool(t)
	store := eventstore.NewPostgresStore(pool)
	ctx := context.Background()
	streamID := "test-load-order-" + t.Name()
	t.Cleanup(func() { cleanupStream(t, pool, streamID) })

	_, err := store.Append(ctx, streamID, 0, []eventstore.UncommittedEvent{
		{EventType: "test.First.v1", Payload: map[string]string{"seq": "1"}},
		{EventType: "test.Second.v1", Payload: map[string]string{"seq": "2"}},
		{EventType: "test.Third.v1", Payload: map[string]string{"seq": "3"}},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := store.LoadStream(ctx, streamID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	for i, e := range events {
		if e.StreamVersion != i+1 {
			t.Errorf("event %d: expected version %d, got %d", i, i+1, e.StreamVersion)
		}
	}
	if events[0].EventType != "test.First.v1" {
		t.Errorf("expected first event type test.First.v1, got %s", events[0].EventType)
	}
}
