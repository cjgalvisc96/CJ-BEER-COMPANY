package muflone

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// The transactional outbox closes the append-then-publish dual-write: in
// durable mode the Postgres event store writes each event's wire message
// into the outbox IN THE SAME TRANSACTION as the stream append, and the
// relay publishes and deletes them. A crash between append and publish can
// no longer lose an event — delivery becomes at-least-once end to end,
// which the system's pervasive idempotency is built to absorb.

// OutboxRelay drains the outbox onto the service bus.
type OutboxRelay struct {
	db       *sql.DB
	bus      *ServiceBus
	interval time.Duration
	logger   *slog.Logger
}

func NewOutboxRelay(db *sql.DB, bus *ServiceBus, interval time.Duration, logger *slog.Logger) *OutboxRelay {
	return &OutboxRelay{db: db, bus: bus, interval: interval, logger: logger}
}

// Run polls until the context is cancelled. Dispatch failures are logged
// and retried on the next tick — the outbox rows stay put until published.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := r.DispatchPending(ctx); err != nil {
				r.logger.Error("outbox.dispatch_failed", slog.String("error", err.Error()))
			}
		}
	}
}

// DispatchPending publishes one batch. FOR UPDATE SKIP LOCKED makes
// concurrent relays (replicas) claim disjoint rows; rows are deleted only
// after a successful publish, inside the same transaction.
func (r *OutboxRelay) DispatchPending(ctx context.Context) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("outbox: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, topic, payload FROM outbox ORDER BY id LIMIT 100 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0, fmt.Errorf("outbox: %w", err)
	}
	type pendingMessage struct {
		id      int64
		topic   string
		payload []byte
	}
	var pending []pendingMessage
	for rows.Next() {
		var row pendingMessage
		if err := rows.Scan(&row.id, &row.topic, &row.payload); err != nil {
			rows.Close()
			return 0, fmt.Errorf("outbox: %w", err)
		}
		pending = append(pending, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("outbox: %w", err)
	}

	for _, row := range pending {
		if err := r.bus.publisher.Publish(row.topic,
			message.NewMessage(watermill.NewUUID(), row.payload)); err != nil {
			return 0, fmt.Errorf("outbox publish %s: %w", row.topic, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM outbox WHERE id = $1`, row.id); err != nil {
			return 0, fmt.Errorf("outbox: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("outbox: %w", err)
	}
	return len(pending), nil
}
