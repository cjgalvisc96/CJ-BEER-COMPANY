// White-box tests for the retry policy, the transactional-outbox relay,
// and the dead-letter archive/redrive.
package muflone

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingPub always refuses to publish.
type failingPub struct{}

func (failingPub) Publish(string, ...*message.Message) error { return assert.AnError }
func (failingPub) Close() error                              { return nil }

// --- retry policy ------------------------------------------------------------

func retryProbe(failures int, err error) (message.HandlerFunc, *int) {
	calls := 0
	return func(*message.Message) ([]*message.Message, error) {
		calls++
		if calls <= failures {
			return nil, err
		}
		return nil, nil
	}, &calls
}

func newRetryMessage() *message.Message {
	msg := message.NewMessage("test", nil)
	msg.SetContext(context.Background())
	return msg
}

func TestRetryPolicyDistinguishesConcurrencyFromPoison(t *testing.T) {
	// A generic failure is out of budget after genericRetries.
	handler, calls := retryProbe(1000, assert.AnError)
	_, err := retryMiddleware(handler)(newRetryMessage())
	assert.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, genericRetries+1, *calls)

	// A concurrency conflict gets a patient budget: succeeding on attempt
	// 7 would already have been poisoned under the generic policy.
	handler, calls = retryProbe(6, ErrConcurrency)
	_, err = retryMiddleware(handler)(newRetryMessage())
	assert.NoError(t, err)
	assert.Equal(t, 7, *calls)

	// First-try success never sleeps.
	handler, calls = retryProbe(0, nil)
	_, err = retryMiddleware(handler)(newRetryMessage())
	assert.NoError(t, err)
	assert.Equal(t, 1, *calls)
}

func TestRetryStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	msg := message.NewMessage("test", nil)
	msg.SetContext(ctx)
	handler, calls := retryProbe(1000, ErrConcurrency)

	_, err := retryMiddleware(handler)(msg)

	assert.ErrorIs(t, err, ErrConcurrency)
	assert.Equal(t, 1, *calls, "no retries after shutdown")
}

func TestRetryBackoffIsCapped(t *testing.T) {
	assert.Equal(t, baseRetryInterval, retryBackoff(0))
	assert.Equal(t, maxRetryInterval, retryBackoff(30), "large attempts cap")
	assert.Equal(t, maxRetryInterval, retryBackoff(63), "overflow caps too")
}

// --- outbox relay ------------------------------------------------------------

func newRelay(t *testing.T, busClosed bool) (*OutboxRelay, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	bus := NewServiceBus(slog.New(slog.DiscardHandler))
	if busClosed {
		require.NoError(t, bus.Close())
	} else {
		t.Cleanup(func() { _ = bus.Close() })
	}
	return NewOutboxRelay(db, bus, 5*time.Millisecond, slog.New(slog.DiscardHandler)), mock
}

func outboxRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "topic", "payload"})
}

func TestOutboxDispatchPublishesAndDeletes(t *testing.T) {
	relay, mock := newRelay(t, false)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, topic, payload FROM outbox").WillReturnRows(
		outboxRows().
			AddRow(1, "events.test.stub_happened", []byte(`{"name":"a"}`)).
			AddRow(2, "events.test.stub_happened", []byte(`{"name":"b"}`)))
	mock.ExpectExec("DELETE FROM outbox").WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM outbox").WithArgs(int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	dispatched, err := relay.DispatchPending(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, dispatched)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxDispatchErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("begin fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin().WillReturnError(assert.AnError)
		_, err := relay.DispatchPending(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("query fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		_, err := relay.DispatchPending(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("scan fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow("not-an-int", "t", "p"))
		mock.ExpectRollback()
		_, err := relay.DispatchPending(ctx)
		assert.Error(t, err)
	})
	t.Run("iteration fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "t", []byte(`{}`)).RowError(0, assert.AnError))
		mock.ExpectRollback()
		_, err := relay.DispatchPending(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("publish fails, rows stay put", func(t *testing.T) {
		relay, mock := newRelay(t, true) // dead bus
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "events.x", []byte(`{}`)))
		mock.ExpectRollback()
		_, err := relay.DispatchPending(ctx)
		assert.ErrorContains(t, err, "outbox publish")
	})
	t.Run("delete fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "events.x", []byte(`{}`)))
		mock.ExpectExec("DELETE FROM outbox").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		_, err := relay.DispatchPending(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("commit fails", func(t *testing.T) {
		relay, mock := newRelay(t, false)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(outboxRows())
		mock.ExpectCommit().WillReturnError(assert.AnError)
		_, err := relay.DispatchPending(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestOutboxRelayRunLoop(t *testing.T) {
	relay, mock := newRelay(t, false)
	// First tick dispatches (empty batch), second tick fails (covers the
	// error log), then the context stops the loop.
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(outboxRows())
	mock.ExpectCommit()
	mock.ExpectBegin().WillReturnError(assert.AnError)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	relay.Run(ctx)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- dead letters ------------------------------------------------------------

func TestDeadLetterStoreSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewDeadLetterStore(db)

	mock.ExpectExec("INSERT INTO dead_letters").
		WithArgs("commands.x", []byte(`{}`), "boom").
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, store.Save(context.Background(), "commands.x", []byte(`{}`), "boom"))

	mock.ExpectExec("INSERT INTO dead_letters").WillReturnError(assert.AnError)
	assert.ErrorIs(t, store.Save(context.Background(), "commands.x", nil, ""), assert.AnError)
}

func TestRedriveDeadLetters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	publisher := &fakePublisher{}
	logger := slog.New(slog.DiscardHandler)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, topic, payload FROM dead_letters").WillReturnRows(
		outboxRows().AddRow(7, "commands.x", []byte(`{"retry":true}`)))
	mock.ExpectExec("UPDATE dead_letters SET redriven_at").WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	redriven, err := RedriveDeadLetters(ctx, db, publisher, logger)

	require.NoError(t, err)
	assert.Equal(t, 1, redriven)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRedriveDeadLettersErrors(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	ctx := context.Background()
	newDB := func(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		return db, mock
	}

	t.Run("begin fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin().WillReturnError(assert.AnError)
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("query fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("scan fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow("junk", "t", "p"))
		mock.ExpectRollback()
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.Error(t, err)
	})
	t.Run("iteration fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "t", []byte(`{}`)).RowError(0, assert.AnError))
		mock.ExpectRollback()
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("publish fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "t", []byte(`{}`)))
		mock.ExpectRollback()
		_, err := RedriveDeadLetters(ctx, db, failingPub{}, logger)
		assert.ErrorContains(t, err, "redrive publish")
	})
	t.Run("update fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(
			outboxRows().AddRow(1, "t", []byte(`{}`)))
		mock.ExpectExec("UPDATE dead_letters").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("commit fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, topic, payload").WillReturnRows(outboxRows())
		mock.ExpectCommit().WillReturnError(assert.AnError)
		_, err := RedriveDeadLetters(ctx, db, &fakePublisher{}, logger)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestOnDeadLetterUnwrapsPoisonMetadata(t *testing.T) {
	bus := NewServiceBus(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { _ = bus.Close() })
	type archived struct{ topic, reason, payload string }
	received := make(chan archived, 1)
	bus.OnDeadLetter("test.archive", func(_ context.Context, topic string, payload []byte, reason string) error {
		received <- archived{topic: topic, reason: reason, payload: string(payload)}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = bus.Run(ctx) }()
	<-bus.Running()

	msg := message.NewMessage("id", []byte(`{"poison":true}`))
	msg.Metadata.Set(middleware.PoisonedTopicKey, "commands.broken")
	msg.Metadata.Set(middleware.ReasonForPoisonedKey, "kaboom")
	require.NoError(t, bus.publisher.Publish(DeadLetterTopic, msg))

	select {
	case got := <-received:
		assert.Equal(t, "commands.broken", got.topic)
		assert.Equal(t, "kaboom", got.reason)
		assert.Contains(t, got.payload, "poison")
	case <-time.After(3 * time.Second):
		t.Fatal("dead letter never reached the archive handler")
	}
}
