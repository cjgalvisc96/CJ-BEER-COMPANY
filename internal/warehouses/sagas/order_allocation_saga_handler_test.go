package sagas_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

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

type sagaFixture struct {
	store *muflone.InMemoryEventStore
	bus   *muflone.ServiceBus
	saga  *sagas.OrderAllocationSaga
}

func newFixture(t *testing.T, busClosed bool) *sagaFixture {
	t.Helper()
	store := muflone.NewInMemoryEventStore()
	bus := muflone.NewServiceBus(slog.Default())
	if busClosed {
		require.NoError(t, bus.Close())
	} else {
		t.Cleanup(func() { _ = bus.Close() })
	}
	repository := muflone.NewEventStoreRepository(store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil)
	return &sagaFixture{
		store: store,
		bus:   bus,
		saga:  sagas.NewOrderAllocationSaga(repository, store, bus, slog.Default()),
	}
}

func (f *sagaFixture) seed(t *testing.T, orderId uuid.UUID, sagaEvents ...muflone.DomainEvent) {
	t.Helper()
	f.store.Seed(domain.SagaStreamName+"-"+orderId.String(), sagaEvents)
}

func triggerPayload(orderId string, rows string) []byte {
	return []byte(`{"commit_id":"` + uuid.NewString() + `","sales_order_id":"` + orderId + `","rows":[` + rows + `]}`)
}

func rowJSON(beerId string, quantity int) string {
	return fmt.Sprintf(`{"beer_id":%q,"beer_name":"IPA","quantity":%d,"unit_of_measure":"Lt"}`,
		beerId, quantity)
}

func TestTriggerEdgeCases(t *testing.T) {
	fixture := newFixture(t, false)
	ctx := context.Background()

	assert.Error(t, fixture.saga.OnSalesOrderCreated(ctx, []byte(`not json`)),
		"malformed trigger is a poison message")
	assert.NoError(t, fixture.saga.OnSalesOrderCreated(ctx, triggerPayload("not-a-uuid", "")),
		"an unusable order id is logged and acked")

	orderId := uuid.New()
	fixture.seed(t, orderId, events.NewOrderAllocationStarted(orderId, uuid.New(), nil))
	assert.NoError(t, fixture.saga.OnSalesOrderCreated(ctx, triggerPayload(orderId.String(), "")),
		"an already-started saga is idempotent")
}

func TestTriggerWithInvalidBeerIdFailsTheStepDispatch(t *testing.T) {
	fixture := newFixture(t, false)

	err := fixture.saga.OnSalesOrderCreated(context.Background(),
		triggerPayload(uuid.NewString(), rowJSON("not-a-uuid", 5)))

	assert.ErrorContains(t, err, "invalid beer id")
}

func TestTriggerWithNoRowsCompletesImmediately(t *testing.T) {
	fixture := newFixture(t, false)

	err := fixture.saga.OnSalesOrderCreated(context.Background(), triggerPayload(uuid.NewString(), ""))

	assert.NoError(t, err)
}

func TestTriggerSurfacesBusFailure(t *testing.T) {
	fixture := newFixture(t, true)

	err := fixture.saga.OnSalesOrderCreated(context.Background(),
		triggerPayload(uuid.NewString(), rowJSON(uuid.NewString(), 5)))

	assert.Error(t, err)
}

func stepEvent(orderId uuid.UUID, beerId uuid.UUID) events.BeerAvailabilityUpdated {
	return events.NewBeerAvailabilityUpdated(
		sharedkernel.BeerId{Value: beerId}, uuid.New(),
		sharedkernel.BeerName{Value: "IPA"}, customtypes.NewQuantity(70, "Lt"), orderId.String(),
	)
}

func TestStepEventsForUnknownSagasAreIgnored(t *testing.T) {
	fixture := newFixture(t, false)
	ctx := context.Background()

	assert.NoError(t, fixture.saga.OnBeerAvailabilityUpdated(ctx, stepEvent(uuid.New(), uuid.New())),
		"unknown saga: replayed or foreign event")
	assert.NoError(t, fixture.saga.OnBeerAvailabilityUpdated(ctx,
		events.NewBeerAvailabilityUpdated(
			sharedkernel.BeerId{Value: uuid.New()}, uuid.New(),
			sharedkernel.BeerName{Value: "IPA"}, customtypes.NewQuantity(1, "Lt"), "",
		)), "plain production flows carry no order")
	assert.NoError(t, fixture.saga.OnQuantityNotFound(ctx, events.NewQuantityNotFound(
		sharedkernel.BeerId{Value: uuid.New()}, uuid.New(), "not-a-uuid",
		customtypes.NewQuantity(1, "Lt"), customtypes.NewQuantity(0, "Lt"),
	)), "unparseable correlation is ignored")
	assert.NoError(t, fixture.saga.OnAvailabilityCompensated(ctx, events.NewAvailabilityCompensated(
		sharedkernel.BeerId{Value: uuid.New()}, uuid.New(),
		sharedkernel.BeerName{Value: "IPA"}, customtypes.NewQuantity(1, "Lt"), uuid.NewString(),
	)), "compensation for an unknown saga is ignored")
}

func TestRedeliveredStepEventsAreIdempotent(t *testing.T) {
	fixture := newFixture(t, false)
	ctx := context.Background()
	orderId, beerId := uuid.New(), uuid.New()
	row := events.SagaRowData{BeerId: beerId.String(), BeerName: "IPA", Quantity: 5, UnitOfMeasure: "Lt"}
	fixture.seed(t, orderId,
		events.NewOrderAllocationStarted(orderId, uuid.New(), []events.SagaRowData{row}),
		events.NewAllocationStepSucceeded(orderId, uuid.New(), beerId.String()),
		events.NewOrderAllocationCompleted(orderId, uuid.New()),
	)

	assert.NoError(t, fixture.saga.OnBeerAvailabilityUpdated(ctx, stepEvent(orderId, beerId)),
		"a redelivered success changes nothing on a settled saga")
	assert.NoError(t, fixture.saga.OnQuantityNotFound(ctx, events.NewQuantityNotFound(
		sharedkernel.BeerId{Value: beerId}, uuid.New(), orderId.String(),
		customtypes.NewQuantity(5, "Lt"), customtypes.NewQuantity(0, "Lt"),
	)))
	assert.NoError(t, fixture.saga.OnAvailabilityCompensated(ctx, events.NewAvailabilityCompensated(
		sharedkernel.BeerId{Value: beerId}, uuid.New(),
		sharedkernel.BeerName{Value: "IPA"}, customtypes.NewQuantity(5, "Lt"), orderId.String(),
	)))
	assert.Empty(t, fixture.store.Appended(), "no new saga events were committed")
}

// TestCompensationDispatchFailures: rows recorded with an unusable beer id
// or a dead bus surface as errors so the bus redelivers.
func TestCompensationDispatchFailures(t *testing.T) {
	ctx := context.Background()
	quantityNotFoundFor := func(orderId uuid.UUID, beerId uuid.UUID) events.QuantityNotFound {
		return events.NewQuantityNotFound(
			sharedkernel.BeerId{Value: beerId}, uuid.New(), orderId.String(),
			customtypes.NewQuantity(5, "Lt"), customtypes.NewQuantity(0, "Lt"),
		)
	}

	t.Run("unusable beer id in a compensable row", func(t *testing.T) {
		fixture := newFixture(t, false)
		orderId, pendingBeer := uuid.New(), uuid.New()
		badRow := events.SagaRowData{BeerId: "not-a-uuid", BeerName: "Ghost", Quantity: 5, UnitOfMeasure: "Lt"}
		pendingRow := events.SagaRowData{BeerId: pendingBeer.String(), BeerName: "IPA", Quantity: 5, UnitOfMeasure: "Lt"}
		fixture.seed(t, orderId,
			events.NewOrderAllocationStarted(orderId, uuid.New(), []events.SagaRowData{badRow, pendingRow}),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), badRow.BeerId),
		)

		err := fixture.saga.OnQuantityNotFound(ctx, quantityNotFoundFor(orderId, pendingBeer))

		assert.ErrorContains(t, err, "invalid beer id")
	})

	t.Run("dead bus while sending the compensation", func(t *testing.T) {
		fixture := newFixture(t, true)
		orderId, allocatedBeer, pendingBeer := uuid.New(), uuid.New(), uuid.New()
		fixture.seed(t, orderId,
			events.NewOrderAllocationStarted(orderId, uuid.New(), []events.SagaRowData{
				{BeerId: allocatedBeer.String(), BeerName: "A", Quantity: 5, UnitOfMeasure: "Lt"},
				{BeerId: pendingBeer.String(), BeerName: "B", Quantity: 5, UnitOfMeasure: "Lt"},
			}),
			events.NewAllocationStepSucceeded(orderId, uuid.New(), allocatedBeer.String()),
		)

		err := fixture.saga.OnQuantityNotFound(ctx, quantityNotFoundFor(orderId, pendingBeer))

		assert.Error(t, err)
	})
}

// failingRepository wraps the real one and injects storage faults, so the
// runner's hard-error branches (nothing to do with business flow) are
// provable.
type failingRepository struct {
	inner    muflone.Repository[*domain.OrderAllocationSaga]
	failGet  bool
	failSave bool
}

func (r *failingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.OrderAllocationSaga, error) {
	if r.failGet {
		return nil, assert.AnError
	}
	return r.inner.GetByID(ctx, id)
}

func (r *failingRepository) Save(ctx context.Context, saga *domain.OrderAllocationSaga, commitId uuid.UUID) error {
	if r.failSave {
		return assert.AnError
	}
	return r.inner.Save(ctx, saga, commitId)
}

func TestStorageFaultsSurfaceForRedelivery(t *testing.T) {
	ctx := context.Background()
	orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
	rows := []events.SagaRowData{
		{BeerId: beerA.String(), BeerName: "A", Quantity: 5, UnitOfMeasure: "Lt"},
		{BeerId: beerB.String(), BeerName: "B", Quantity: 5, UnitOfMeasure: "Lt"},
	}
	newFaultyFixture := func(t *testing.T, failGet, failSave bool, seed ...muflone.DomainEvent) *sagas.OrderAllocationSaga {
		t.Helper()
		store := muflone.NewInMemoryEventStore()
		store.Seed(domain.SagaStreamName+"-"+orderId.String(), seed)
		bus := muflone.NewServiceBus(slog.Default())
		t.Cleanup(func() { _ = bus.Close() })
		repository := &failingRepository{
			inner:    muflone.NewEventStoreRepository(store, domain.NewOrderAllocationSaga, domain.SagaStreamName, nil),
			failGet:  failGet,
			failSave: failSave,
		}
		return sagas.NewOrderAllocationSaga(repository, store, bus, slog.Default())
	}
	started := events.NewOrderAllocationStarted(orderId, uuid.New(), rows)
	allocatedA := events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String())
	failedB := events.NewAllocationStepFailed(orderId, uuid.New(), beerB.String(), "shortage")

	assert.Error(t, newFaultyFixture(t, true, false).
		OnSalesOrderCreated(ctx, triggerPayload(orderId.String(), rowJSON(beerA.String(), 5))),
		"trigger: existence check fails hard")
	assert.Error(t, newFaultyFixture(t, false, true).
		OnSalesOrderCreated(ctx, triggerPayload(orderId.String(), rowJSON(beerA.String(), 5))),
		"trigger: saga cannot be persisted")
	assert.Error(t, newFaultyFixture(t, true, false, started).
		OnBeerAvailabilityUpdated(ctx, stepEvent(orderId, beerA)),
		"step: saga cannot be loaded")
	assert.Error(t, newFaultyFixture(t, false, true, started).
		OnBeerAvailabilityUpdated(ctx, stepEvent(orderId, beerA)),
		"step: saga cannot be persisted")
	assert.Error(t, newFaultyFixture(t, false, true, started, allocatedA).
		OnQuantityNotFound(ctx, events.NewQuantityNotFound(
			sharedkernel.BeerId{Value: beerB}, uuid.New(), orderId.String(),
			customtypes.NewQuantity(5, "Lt"), customtypes.NewQuantity(0, "Lt"))),
		"failure: saga cannot be persisted")
	assert.Error(t, newFaultyFixture(t, false, true, started, allocatedA, failedB).
		OnAvailabilityCompensated(ctx, events.NewAvailabilityCompensated(
			sharedkernel.BeerId{Value: beerA}, uuid.New(),
			sharedkernel.BeerName{Value: "A"}, customtypes.NewQuantity(5, "Lt"), orderId.String())),
		"compensation: saga cannot be persisted")
}

// TestCompensationCompletesAndRejects covers the full backward-recovery
// path against a live in-process fixture (open bus): two rows were already
// allocated when the third fails, so the first compensation leaves the
// saga still compensating and only the second rejects the order.
func TestCompensationCompletesAndRejects(t *testing.T) {
	fixture := newFixture(t, false)
	ctx := context.Background()
	orderId, beerA, beerB, beerC := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	fixture.seed(t, orderId,
		events.NewOrderAllocationStarted(orderId, uuid.New(), []events.SagaRowData{
			{BeerId: beerA.String(), BeerName: "A", Quantity: 5, UnitOfMeasure: "Lt"},
			{BeerId: beerB.String(), BeerName: "B", Quantity: 5, UnitOfMeasure: "Lt"},
			{BeerId: beerC.String(), BeerName: "C", Quantity: 5, UnitOfMeasure: "Lt"},
		}),
		events.NewAllocationStepSucceeded(orderId, uuid.New(), beerA.String()),
		events.NewAllocationStepSucceeded(orderId, uuid.New(), beerB.String()),
	)
	compensated := func(beerId uuid.UUID, name string) events.AvailabilityCompensated {
		return events.NewAvailabilityCompensated(
			sharedkernel.BeerId{Value: beerId}, uuid.New(),
			sharedkernel.BeerName{Value: name}, customtypes.NewQuantity(5, "Lt"), orderId.String())
	}

	require.NoError(t, fixture.saga.OnQuantityNotFound(ctx, events.NewQuantityNotFound(
		sharedkernel.BeerId{Value: beerC}, uuid.New(), orderId.String(),
		customtypes.NewQuantity(5, "Lt"), customtypes.NewQuantity(0, "Lt"))))
	require.NoError(t, fixture.saga.OnAvailabilityCompensated(ctx, compensated(beerA, "A")),
		"first compensation: saga keeps compensating")
	require.NoError(t, fixture.saga.OnAvailabilityCompensated(ctx, compensated(beerB, "B")),
		"last compensation: saga rejects the order")
}

// TestNextAllocationBusFailure: the saga advanced but the next step's
// command could not be sent — the error propagates for redelivery.
func TestNextAllocationBusFailure(t *testing.T) {
	fixture := newFixture(t, true)
	ctx := context.Background()
	orderId, beerA, beerB := uuid.New(), uuid.New(), uuid.New()
	fixture.seed(t, orderId,
		events.NewOrderAllocationStarted(orderId, uuid.New(), []events.SagaRowData{
			{BeerId: beerA.String(), BeerName: "A", Quantity: 5, UnitOfMeasure: "Lt"},
			{BeerId: beerB.String(), BeerName: "B", Quantity: 5, UnitOfMeasure: "Lt"},
		}),
	)

	err := fixture.saga.OnBeerAvailabilityUpdated(ctx, stepEvent(orderId, beerA))

	assert.Error(t, err)
}
