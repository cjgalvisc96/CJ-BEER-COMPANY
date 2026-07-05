package muflone

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// DeadLetterStore archives poison messages durably (durable mode) so an
// operator can inspect and RE-DRIVE them after fixing the cause — parked,
// never lost (book Ch. 12).
type DeadLetterStore struct {
	db *sql.DB
}

func NewDeadLetterStore(db *sql.DB) *DeadLetterStore {
	return &DeadLetterStore{db: db}
}

func (s *DeadLetterStore) Save(ctx context.Context, topic string, payload []byte, reason string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO dead_letters (topic, payload, reason) VALUES ($1, $2, $3)`,
		topic, payload, reason,
	); err != nil {
		return fmt.Errorf("archive dead letter: %w", err)
	}
	return nil
}

// OnDeadLetter subscribes a handler to the dead-letter topic, unwrapping
// the poison metadata (original topic + failure reason).
func (b *ServiceBus) OnDeadLetter(name string, handler func(ctx context.Context, topic string, payload []byte, reason string) error) {
	b.subscribe(name, DeadLetterTopic, func(msg *message.Message) error {
		return handler(msg.Context(),
			msg.Metadata.Get(middleware.PoisonedTopicKey),
			msg.Payload,
			msg.Metadata.Get(middleware.ReasonForPoisonedKey))
	})
}

// RedriveDeadLetters republishes every un-redriven dead letter to its
// original topic and stamps it. SKIP LOCKED keeps concurrent redrives from
// double-claiming.
func RedriveDeadLetters(ctx context.Context, db *sql.DB, publisher message.Publisher, logger *slog.Logger) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("redrive: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, topic, payload FROM dead_letters
		  WHERE redriven_at IS NULL ORDER BY id FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0, fmt.Errorf("redrive: %w", err)
	}
	type deadLetter struct {
		id      int64
		topic   string
		payload []byte
	}
	var parked []deadLetter
	for rows.Next() {
		var row deadLetter
		if err := rows.Scan(&row.id, &row.topic, &row.payload); err != nil {
			rows.Close()
			return 0, fmt.Errorf("redrive: %w", err)
		}
		parked = append(parked, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("redrive: %w", err)
	}

	for _, letter := range parked {
		if err := publisher.Publish(letter.topic,
			message.NewMessage(watermill.NewUUID(), letter.payload)); err != nil {
			return 0, fmt.Errorf("redrive publish %s: %w", letter.topic, err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE dead_letters SET redriven_at = now() WHERE id = $1`, letter.id); err != nil {
			return 0, fmt.Errorf("redrive: %w", err)
		}
		logger.Info("dead_letter.redriven", slog.String("topic", letter.topic))
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("redrive: %w", err)
	}
	return len(parked), nil
}
