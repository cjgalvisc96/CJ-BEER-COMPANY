package muflone

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStoreWithMock(t *testing.T) (*PostgresEventStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	registry := NewEventRegistry()
	RegisterEvent[stubEvent](registry)
	return NewPostgresEventStore(db, registry), mock
}

func storedPayload(t *testing.T, event stubEvent) []byte {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return payload
}

func TestPostgresReadStream(t *testing.T) {
	store, mock := newStoreWithMock(t)
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "one"}
	mock.ExpectQuery("SELECT version, commit_id, event_type, payload, occurred_at").
		WithArgs("Stub-1").
		WillReturnRows(sqlmock.NewRows(
			[]string{"version", "commit_id", "event_type", "payload", "occurred_at"}).
			AddRow(1, event.CommitID(), event.MessageName(), storedPayload(t, event), time.Now()))

	stored, err := store.ReadStream(context.Background(), "Stub-1")

	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, event, stored[0].Event)
	assert.Equal(t, 1, stored[0].Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresReadStreamErrors(t *testing.T) {
	t.Run("query fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectQuery("SELECT version").WillReturnError(assert.AnError)

		_, err := store.ReadStream(context.Background(), "Stub-1")
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("scan fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows(
			[]string{"version", "commit_id", "event_type", "payload", "occurred_at"}).
			AddRow("not-an-int", "x", "y", "{}", "z"))

		_, err := store.ReadStream(context.Background(), "Stub-1")
		assert.ErrorContains(t, err, "scan stream")
	})

	t.Run("unknown event type", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows(
			[]string{"version", "commit_id", "event_type", "payload", "occurred_at"}).
			AddRow(1, uuid.New(), "test.never_registered", []byte(`{}`), time.Now()))

		_, err := store.ReadStream(context.Background(), "Stub-1")
		assert.ErrorContains(t, err, "unknown event type")
	})

	t.Run("iteration fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "x"}
		mock.ExpectQuery("SELECT version").WillReturnRows(sqlmock.NewRows(
			[]string{"version", "commit_id", "event_type", "payload", "occurred_at"}).
			AddRow(1, event.CommitID(), event.MessageName(), storedPayload(t, event), time.Now()).
			RowError(0, assert.AnError))

		_, err := store.ReadStream(context.Background(), "Stub-1")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func headQuery(mock sqlmock.Sqlmock, head int) {
	mock.ExpectQuery("SELECT coalesce").WithArgs("Stub-1").
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(head))
}

func TestPostgresAppend(t *testing.T) {
	store, mock := newStoreWithMock(t)
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "one"}
	mock.ExpectBegin()
	headQuery(mock, 0)
	mock.ExpectExec("INSERT INTO events").
		WithArgs("Stub-1", 1, sqlmock.AnyArg(), event.MessageName(), storedPayload(t, event)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := store.Append(context.Background(), "Stub-1", 0, uuid.New(), []DomainEvent{event})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresAppendErrors(t *testing.T) {
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "one"}
	events := []DomainEvent{event}
	ctx := context.Background()

	t.Run("begin fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin().WillReturnError(assert.AnError)
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), assert.AnError)
	})

	t.Run("head query fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT coalesce").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), assert.AnError)
	})

	t.Run("version mismatch", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		headQuery(mock, 7)
		mock.ExpectRollback()
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), ErrConcurrency)
	})

	t.Run("marshal fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		headQuery(mock, 0)
		mock.ExpectRollback()
		bad := badPayloadEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New())}
		assert.ErrorContains(t,
			store.Append(ctx, "Stub-1", 0, uuid.New(), []DomainEvent{bad}), "marshal event")
	})

	t.Run("unique violation maps to concurrency", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		headQuery(mock, 0)
		mock.ExpectExec("INSERT INTO events").
			WillReturnError(&pgconn.PgError{Code: pgUniqueViolation})
		mock.ExpectRollback()
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), ErrConcurrency)
	})

	t.Run("other insert error surfaces", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		headQuery(mock, 0)
		mock.ExpectExec("INSERT INTO events").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), assert.AnError)
	})

	t.Run("commit fails", func(t *testing.T) {
		store, mock := newStoreWithMock(t)
		mock.ExpectBegin()
		headQuery(mock, 0)
		mock.ExpectExec("INSERT INTO events").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(assert.AnError)
		assert.ErrorIs(t, store.Append(ctx, "Stub-1", 0, uuid.New(), events), assert.AnError)
	})
}
