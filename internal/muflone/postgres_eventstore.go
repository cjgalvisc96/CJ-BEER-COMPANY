package muflone

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgUniqueViolation = "23505"

// PostgresEventStore is the durable EventStore (the book's Chapter 8 step:
// the schema lives in migrations/ — an append-only "events" table keyed by
// (stream_id, version)). Optimistic concurrency is enforced twice: by
// comparing the stream head inside the transaction and, against races the
// read cannot see, by the primary key itself.
type PostgresEventStore struct {
	db       *sql.DB
	registry *EventRegistry
}

func NewPostgresEventStore(db *sql.DB, registry *EventRegistry) *PostgresEventStore {
	return &PostgresEventStore{db: db, registry: registry}
}

func (s *PostgresEventStore) ReadStream(ctx context.Context, streamID string) ([]StoredEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT version, commit_id, event_type, payload, occurred_at
		   FROM events WHERE stream_id = $1 ORDER BY version`, streamID)
	if err != nil {
		return nil, fmt.Errorf("read stream %s: %w", streamID, err)
	}
	defer rows.Close()

	var stored []StoredEvent
	for rows.Next() {
		var record StoredEvent
		var eventType string
		var payload []byte
		record.StreamID = streamID
		if err := rows.Scan(&record.Version, &record.CommitID, &eventType, &payload, &record.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan stream %s: %w", streamID, err)
		}
		event, err := s.registry.Deserialize(eventType, payload)
		if err != nil {
			return nil, fmt.Errorf("stream %s version %d: %w", streamID, record.Version, err)
		}
		record.Event = event
		stored = append(stored, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stream %s: %w", streamID, err)
	}
	return stored, nil
}

// ListStreams returns the stream ids under a prefix, sorted.
func (s *PostgresEventStore) ListStreams(ctx context.Context, streamPrefix string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT stream_id FROM events WHERE stream_id LIKE $1 ORDER BY stream_id`,
		streamPrefix+"-%")
	if err != nil {
		return nil, fmt.Errorf("list streams %s: %w", streamPrefix, err)
	}
	defer rows.Close()

	var streams []string
	for rows.Next() {
		var streamID string
		if err := rows.Scan(&streamID); err != nil {
			return nil, fmt.Errorf("list streams %s: %w", streamPrefix, err)
		}
		streams = append(streams, streamID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list streams %s: %w", streamPrefix, err)
	}
	return streams, nil
}

func (s *PostgresEventStore) Append(
	ctx context.Context,
	streamID string,
	expectedVersion int,
	commitID uuid.UUID,
	events []DomainEvent,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("append to %s: %w", streamID, err)
	}
	defer func() { _ = tx.Rollback() }()

	var head int
	if err := tx.QueryRowContext(ctx,
		`SELECT coalesce(max(version), 0) FROM events WHERE stream_id = $1`, streamID,
	).Scan(&head); err != nil {
		return fmt.Errorf("append to %s: %w", streamID, err)
	}
	if head != expectedVersion {
		return fmt.Errorf("%w: stream %s is at version %d, expected %d",
			ErrConcurrency, streamID, head, expectedVersion)
	}

	for i, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event %s: %w", event.MessageName(), err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO events (stream_id, version, commit_id, event_type, payload)
			 VALUES ($1, $2, $3, $4, $5)`,
			streamID, expectedVersion+i+1, commitID, event.MessageName(), payload,
		); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return fmt.Errorf("%w: stream %s version %d already written",
					ErrConcurrency, streamID, expectedVersion+i+1)
			}
			return fmt.Errorf("append to %s: %w", streamID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("append to %s: %w", streamID, err)
	}
	return nil
}
