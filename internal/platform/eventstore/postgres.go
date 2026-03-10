package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/database"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL-backed event store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Append persists events to a stream, enforcing optimistic concurrency via expected version.
func (s *PostgresStore) Append(ctx context.Context, streamID string, expectedVersion int, events []UncommittedEvent) ([]StoredEvent, error) {
	var stored []StoredEvent

	err := database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		for i, e := range events {
			version := expectedVersion + 1 + i

			raw, err := json.Marshal(e.Payload)
			if err != nil {
				return fmt.Errorf("marshal event payload: %w", err)
			}

			var se StoredEvent
			err = tx.QueryRow(ctx, `
				INSERT INTO events (stream_id, stream_version, event_type, payload, causation_id, correlation_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id, stream_id, stream_version, event_type, payload, causation_id, correlation_id, recorded_at
			`, streamID, version, e.EventType, raw, e.CausationID, e.CorrelationID).Scan(
				&se.ID, &se.StreamID, &se.StreamVersion, &se.EventType,
				&se.Payload, &se.CausationID, &se.CorrelationID, &se.RecordedAt,
			)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return ErrConcurrencyConflict
				}
				return fmt.Errorf("insert event: %w", err)
			}

			stored = append(stored, se)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return stored, nil
}

// LoadStream returns all events for a stream, ordered by version ascending.
func (s *PostgresStore) LoadStream(ctx context.Context, streamID string) ([]StoredEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, stream_id, stream_version, event_type, payload, causation_id, correlation_id, recorded_at
		FROM events
		WHERE stream_id = $1
		ORDER BY stream_version ASC
	`, streamID)
	if err != nil {
		return nil, fmt.Errorf("query stream events: %w", err)
	}
	defer rows.Close()

	var events []StoredEvent
	for rows.Next() {
		var se StoredEvent
		if err := rows.Scan(
			&se.ID, &se.StreamID, &se.StreamVersion, &se.EventType,
			&se.Payload, &se.CausationID, &se.CorrelationID, &se.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, se)
	}
	return events, rows.Err()
}
