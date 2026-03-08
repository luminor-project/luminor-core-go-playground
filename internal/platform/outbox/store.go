package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Event struct {
	ID          string
	EventType   string
	Payload     json.RawMessage
	AvailableAt time.Time
	ProcessedAt *time.Time
	Attempts    int
	LastError   *string
	CreatedAt   time.Time
}

type Store interface {
	Enqueue(ctx context.Context, eventType string, payload any) error
	GetPending(ctx context.Context, limit int) ([]Event, error)
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, err error, retryAfter time.Duration) error
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Enqueue(ctx context.Context, eventType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO outbox_events (id, event_type, payload)
		VALUES ($1, $2, $3)
	`, uuid.New().String(), eventType, raw)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetPending(ctx context.Context, limit int) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, payload, available_at, processed_at, attempts, last_error, created_at
		FROM outbox_events
		WHERE processed_at IS NULL AND available_at <= now()
		ORDER BY created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(
			&ev.ID,
			&ev.EventType,
			&ev.Payload,
			&ev.AvailableAt,
			&ev.ProcessedAt,
			&ev.Attempts,
			&ev.LastError,
			&ev.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (s *PostgresStore) MarkProcessed(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET processed_at = now(), last_error = NULL
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("mark outbox event processed: %w", err)
	}
	return nil
}

func (s *PostgresStore) MarkFailed(ctx context.Context, id string, err error, retryAfter time.Duration) error {
	_, updateErr := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
		    last_error = $2,
		    available_at = now() + $3::interval
		WHERE id = $1
	`, id, err.Error(), durationAsSQLInterval(retryAfter))
	if updateErr != nil {
		return fmt.Errorf("mark outbox event failed: %w", updateErr)
	}
	return nil
}

func durationAsSQLInterval(d time.Duration) string {
	return fmt.Sprintf("%f seconds", d.Seconds())
}
