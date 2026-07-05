// Tests for the durable-execution duties (book Ch. 12): boot-time resume
// and the step-timeout watchdog.
package sagas_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sagas"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

var farFuture = time.Now().Add(24 * time.Hour)

func startedWith(orderId uuid.UUID, rows ...events.SagaRowData) muflone.DomainEvent {
	return events.NewOrderAllocationStarted(orderId, uuid.New(), rows)
}

func row(beerId uuid.UUID, name string) events.SagaRowData {
	return events.SagaRowData{BeerId: beerId.String(), BeerName: name, Quantity: 5, UnitOfMeasure: "Lt"}
}

func TestResumeInFlightRedrivesUnfinishedSagas(t *testing.T) {
	fixture := newFixture(t, false)
	ctx := context.Background()

	// A running saga, a compensating saga, a settled saga, and a stream
	// with an unparseable id — only the first two are re-driven.
	runningOrder, beerA := uuid.New(), uuid.New()
	fixture.seed(t, runningOrder, startedWith(runningOrder, row(beerA, "A")))

	compensatingOrder, beerB, beerC := uuid.New(), uuid.New(), uuid.New()
	fixture.seed(t, compensatingOrder,
		startedWith(compensatingOrder, row(beerB, "B"), row(beerC, "C")),
		events.NewAllocationStepSucceeded(compensatingOrder, uuid.New(), beerB.String()),
		events.NewAllocationStepFailed(compensatingOrder, uuid.New(), beerC.String(), "shortage"),
	)

	settledOrder := uuid.New()
	fixture.seed(t, settledOrder,
		startedWith(settledOrder, row(uuid.New(), "Z")),
		events.NewOrderAllocationRejected(settledOrder, uuid.New(), "settled"),
	)

	fixture.store.Seed(domain.SagaStreamName+"-not-a-uuid", []muflone.DomainEvent{
		startedWith(uuid.New()),
	})

	assert.NoError(t, fixture.saga.ResumeInFlight(ctx))
}

func TestResumeInFlightWithoutListerIsANoOp(t *testing.T) {
	store := muflone.NewInMemoryEventStore()
	// listlessStore hides ListStreams: resume degrades to a logged no-op.
	repository := muflone.NewEventStoreRepository(
		listlessStore{store}, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil)
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	saga := sagas.NewOrderAllocationSaga(repository, listlessStore{store}, bus, slog.Default())

	assert.NoError(t, saga.ResumeInFlight(context.Background()))
	assert.NoError(t, saga.TimeoutInFlight(context.Background(), farFuture))
}

type listlessStore struct{ inner muflone.EventStore }

func (s listlessStore) ReadStream(ctx context.Context, streamID string) ([]muflone.StoredEvent, error) {
	return s.inner.ReadStream(ctx, streamID)
}

func (s listlessStore) Append(ctx context.Context, streamID string, expectedVersion int, commitID uuid.UUID, events []muflone.DomainEvent) error {
	return s.inner.Append(ctx, streamID, expectedVersion, commitID, events)
}

type failingLister struct{ muflone.EventStore }

func (failingLister) ListStreams(context.Context, string) ([]string, error) {
	return nil, assert.AnError
}

func TestResumeInFlightSurfacesStoreFailures(t *testing.T) {
	store := muflone.NewInMemoryEventStore()
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })

	// Lister fails.
	broken := sagas.NewOrderAllocationSaga(
		muflone.NewEventStoreRepository(store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil),
		failingLister{store}, bus, slog.Default())
	assert.ErrorIs(t, broken.ResumeInFlight(context.Background()), assert.AnError)
	assert.ErrorIs(t, broken.TimeoutInFlight(context.Background(), farFuture), assert.AnError)

	// Loading a listed saga fails hard.
	orderId := uuid.New()
	store.Seed(domain.SagaStreamName+"-"+orderId.String(),
		[]muflone.DomainEvent{startedWith(orderId, row(uuid.New(), "A"))})
	faulty := sagas.NewOrderAllocationSaga(
		&failingRepository{failGet: true}, store, bus, slog.Default())
	assert.ErrorIs(t, faulty.ResumeInFlight(context.Background()), assert.AnError)
}

// flakyReadStore fails ReadStream after the first read of each stream —
// the repository load succeeds, the saga's activity lookup fails.
type flakyReadStore struct {
	*muflone.InMemoryEventStore
	reads map[string]int
}

func (s *flakyReadStore) ReadStream(ctx context.Context, streamID string) ([]muflone.StoredEvent, error) {
	s.reads[streamID]++
	if s.reads[streamID] > 1 {
		return nil, assert.AnError
	}
	return s.InMemoryEventStore.ReadStream(ctx, streamID)
}

func TestTimeoutInFlightSurfacesActivityLookupFailures(t *testing.T) {
	store := &flakyReadStore{InMemoryEventStore: muflone.NewInMemoryEventStore(), reads: map[string]int{}}
	orderId := uuid.New()
	store.Seed(domain.SagaStreamName+"-"+orderId.String(),
		[]muflone.DomainEvent{startedWith(orderId, row(uuid.New(), "A"))})
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })
	saga := sagas.NewOrderAllocationSaga(
		muflone.NewEventStoreRepository(store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil),
		store, bus, slog.Default())

	assert.ErrorIs(t, saga.TimeoutInFlight(context.Background(), farFuture), assert.AnError)
}

func TestTimeoutInFlightFailsStaleSteps(t *testing.T) {
	ctx := context.Background()

	t.Run("stale running saga with allocated rows compensates", func(t *testing.T) {
		fixture := newFixture(t, false)
		orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
		// Seeded events carry a zero OccurredAt — maximally stale.
		fixture.seed(t, orderId,
			startedWith(orderId, row(beerA, "A"), row(beerB, "B")),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
		)

		require.NoError(t, fixture.saga.TimeoutInFlight(ctx, farFuture))

		saga, err := muflone.NewEventStoreRepository(
			fixture.store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil,
		).GetByID(ctx, orderId)
		require.NoError(t, err)
		assert.Equal(t, domain.SagaCompensating, saga.Status())
		assert.Equal(t, "allocation step timed out", saga.Reason())
	})

	t.Run("stale running saga with nothing allocated rejects", func(t *testing.T) {
		fixture := newFixture(t, false)
		orderId, beerA := uuid.New(), uuid.New()
		fixture.seed(t, orderId, startedWith(orderId, row(beerA, "A")))

		require.NoError(t, fixture.saga.TimeoutInFlight(ctx, farFuture))

		saga, err := muflone.NewEventStoreRepository(
			fixture.store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil,
		).GetByID(ctx, orderId)
		require.NoError(t, err)
		assert.Equal(t, domain.SagaRejected, saga.Status())
	})

	t.Run("stale compensating saga is re-driven, not failed again", func(t *testing.T) {
		fixture := newFixture(t, false)
		orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
		fixture.seed(t, orderId,
			startedWith(orderId, row(beerA, "A"), row(beerB, "B")),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
			events.NewAllocationStepFailed(orderId, uuid.New(), beerB.String(), "shortage"),
		)

		require.NoError(t, fixture.saga.TimeoutInFlight(ctx, farFuture))
		assert.Empty(t, fixture.store.Appended(), "re-driving records nothing new")
	})

	t.Run("fresh sagas are left alone", func(t *testing.T) {
		fixture := newFixture(t, false)
		orderId, beerA := uuid.New(), uuid.New()
		// Appended (not seeded) events carry OccurredAt = now → fresh.
		require.NoError(t, fixture.store.Append(ctx,
			domain.SagaStreamName+"-"+orderId.String(), 0, uuid.New(),
			[]muflone.DomainEvent{startedWith(orderId, row(beerA, "A"))}))

		require.NoError(t, fixture.saga.TimeoutInFlight(ctx, time.Now().Add(-time.Hour)))

		saga, err := muflone.NewEventStoreRepository(
			fixture.store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil,
		).GetByID(ctx, orderId)
		require.NoError(t, err)
		assert.Equal(t, domain.SagaRunning, saga.Status())
	})

	t.Run("save failure surfaces", func(t *testing.T) {
		store := muflone.NewInMemoryEventStore()
		orderId, beerA := uuid.New(), uuid.New()
		store.Seed(domain.SagaStreamName+"-"+orderId.String(),
			[]muflone.DomainEvent{startedWith(orderId, row(beerA, "A"))})
		bus := muflone.NewServiceBus(slog.Default())
		t.Cleanup(func() { _ = bus.Close() })
		saga := sagas.NewOrderAllocationSaga(
			&failingRepository{
				inner: muflone.NewEventStoreRepository(
					store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil),
				failSave: true,
			}, store, bus, slog.Default())

		assert.ErrorIs(t, saga.TimeoutInFlight(ctx, farFuture), assert.AnError)
	})

	t.Run("dead bus while compensating after a timeout", func(t *testing.T) {
		fixture := newFixture(t, true)
		orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
		fixture.seed(t, orderId,
			startedWith(orderId, row(beerA, "A"), row(beerB, "B")),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
		)

		assert.Error(t, fixture.saga.TimeoutInFlight(ctx, farFuture))
	})

	t.Run("dead bus while re-driving a stale compensating saga", func(t *testing.T) {
		fixture := newFixture(t, true)
		orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
		fixture.seed(t, orderId,
			startedWith(orderId, row(beerA, "A"), row(beerB, "B")),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
			events.NewAllocationStepFailed(orderId, uuid.New(), beerB.String(), "shortage"),
		)

		assert.Error(t, fixture.saga.TimeoutInFlight(ctx, farFuture))
	})

	t.Run("dead bus while publishing an immediate rejection", func(t *testing.T) {
		fixture := newFixture(t, true)
		orderId, beerA := uuid.New(), uuid.New()
		fixture.seed(t, orderId, startedWith(orderId, row(beerA, "A")))

		assert.Error(t, fixture.saga.TimeoutInFlight(ctx, farFuture))
	})
}

func TestResumeRedrivesCompensationsToTheBus(t *testing.T) {
	// A compensating saga resumed against a DEAD bus surfaces the failure.
	fixture := newFixture(t, true)
	orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
	fixture.seed(t, orderId,
		startedWith(orderId, row(beerA, "A"), row(beerB, "B")),
		events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
		events.NewAllocationStepFailed(orderId, uuid.New(), beerB.String(), "shortage"),
	)

	assert.Error(t, fixture.saga.ResumeInFlight(context.Background()))
}

// The full resume story, end to end in-process: a saga is interrupted
// mid-flight (its allocation already landed on the Availability stream but
// the saga never saw the outcome), the process "restarts", ResumeInFlight
// re-drives the step, and the idempotent Availability re-emits the fact
// without double-allocating.
func TestResumeAfterCrashDoesNotDoubleAllocate(t *testing.T) {
	ctx := context.Background()
	store := muflone.NewInMemoryEventStore()
	orderId, beerId := uuid.New(), uuid.New()

	// The warehouse allocated 30 of 100 for this order...
	beer := sharedkernel.BeerId{Value: beerId}
	name := sharedkernel.BeerName{Value: "IPA"}
	store.Seed("Availability-"+beerId.String(), []muflone.DomainEvent{
		events.NewAvailabilityUpdatedDueToProductionOrder(beer, uuid.New(), name, customtypes.NewQuantity(100, "Lt")),
		events.NewBeerAvailabilityUpdated(beer, uuid.New(), name, customtypes.NewQuantity(70, "Lt"), orderId.String()),
	})
	// ...but the saga crashed before observing it.
	store.Seed(domain.SagaStreamName+"-"+orderId.String(), []muflone.DomainEvent{
		startedWith(orderId, row(beerId, "IPA")),
	})

	availabilityRepo := muflone.NewEventStoreRepository(store, domain.NewAvailability, domain.StreamName, nil)
	availability, err := availabilityRepo.GetByID(ctx, beerId)
	require.NoError(t, err)

	// The re-driven command is idempotent: quantity stays 70.
	require.NoError(t, availability.UpdateDueToSalesOrder(uuid.New(), customtypes.NewQuantity(30, "Lt"), orderId.String()))
	require.NoError(t, availabilityRepo.Save(ctx, availability, uuid.New()))

	reloaded, err := availabilityRepo.GetByID(ctx, beerId)
	require.NoError(t, err)
	replayed := reloaded.UncommittedEvents()
	assert.Empty(t, replayed)
	stored, err := store.ReadStream(ctx, "Availability-"+beerId.String())
	require.NoError(t, err)
	last, ok := stored[len(stored)-1].Event.(events.BeerAvailabilityUpdated)
	require.True(t, ok)
	assert.Equal(t, 70, last.Quantity.Value, "re-emitted fact, unchanged quantity")
}
