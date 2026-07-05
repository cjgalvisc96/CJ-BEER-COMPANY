// White-box tests for stream listing and the dead-letter queue.
package muflone

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryListStreams(t *testing.T) {
	store := NewInMemoryEventStore()
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "x"}
	store.Seed("Saga-1", []DomainEvent{event})
	store.Seed("Saga-2", []DomainEvent{event})
	store.Seed("Other-1", []DomainEvent{event})

	streams, err := store.ListStreams(context.Background(), "Saga")

	require.NoError(t, err)
	assert.Equal(t, []string{"Saga-1", "Saga-2"}, streams)

	empty, err := store.ListStreams(context.Background(), "Nothing")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestPostgresListStreams(t *testing.T) {
	newStore := func(t *testing.T) (*PostgresEventStore, sqlmock.Sqlmock) {
		t.Helper()
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		return NewPostgresEventStore(db, NewEventRegistry()), mock
	}
	ctx := context.Background()

	t.Run("happy", func(t *testing.T) {
		store, mock := newStore(t)
		mock.ExpectQuery("SELECT DISTINCT stream_id").WithArgs("Saga-%").
			WillReturnRows(sqlmock.NewRows([]string{"stream_id"}).AddRow("Saga-1").AddRow("Saga-2"))

		streams, err := store.ListStreams(ctx, "Saga")

		require.NoError(t, err)
		assert.Equal(t, []string{"Saga-1", "Saga-2"}, streams)
	})
	t.Run("query fails", func(t *testing.T) {
		store, mock := newStore(t)
		mock.ExpectQuery("SELECT DISTINCT stream_id").WillReturnError(assert.AnError)
		_, err := store.ListStreams(ctx, "Saga")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("scan fails", func(t *testing.T) {
		store, mock := newStore(t)
		mock.ExpectQuery("SELECT DISTINCT stream_id").
			WillReturnRows(sqlmock.NewRows([]string{"stream_id"}).AddRow(nil))
		_, err := store.ListStreams(ctx, "Saga")
		assert.Error(t, err)
	})
	t.Run("iteration fails", func(t *testing.T) {
		store, mock := newStore(t)
		mock.ExpectQuery("SELECT DISTINCT stream_id").
			WillReturnRows(sqlmock.NewRows([]string{"stream_id"}).AddRow("Saga-1").
				RowError(0, assert.AnError))
		_, err := store.ListStreams(ctx, "Saga")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

type alwaysFailingHandler struct{}

func (alwaysFailingHandler) Handle(context.Context, stubCommand) error {
	return errors.New("permanently broken")
}

// TestDeadLetterQueueParksPoisonMessages: a message that keeps failing
// after the retries lands on the dead-letter topic (book Ch. 12) — parked
// and logged, never lost, never blocking the bus.
func TestDeadLetterQueueParksPoisonMessages(t *testing.T) {
	bus := NewServiceBus(slog.New(slog.DiscardHandler))
	RegisterCommandHandler[stubCommand](bus, alwaysFailingHandler{})

	deadLetters := make(chan string, 1)
	bus.subscribe("test.dead_letter_probe", DeadLetterTopic, func(msg *message.Message) error {
		deadLetters <- string(msg.Payload)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = bus.Run(ctx) }()
	<-bus.Running()

	command := stubCommand{CommandBase: NewCommandBase(uuid.New(), uuid.New()), Name: "poison"}
	require.NoError(t, bus.Send(ctx, command))

	select {
	case payload := <-deadLetters:
		assert.Contains(t, payload, "poison")
	case <-time.After(5 * time.Second):
		t.Fatal("poison message never reached the dead-letter queue")
	}
	require.NoError(t, bus.Close())
}
