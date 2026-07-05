package warehouses_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

// TestRebuildReadModel replays a beer's full history — production,
// allocation, refusal, compensation — and the projection lands on the
// final cumulative quantity.
func TestRebuildReadModel(t *testing.T) {
	store := muflone.NewInMemoryEventStore()
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "BrewUp IPA"}
	commitId := uuid.New()
	orderId := uuid.NewString()
	store.Seed(domain.StreamName+"-"+beerId.Value.String(), []muflone.DomainEvent{
		events.NewAvailabilityUpdatedDueToProductionOrder(beerId, commitId, beerName, customtypes.NewQuantity(100, "Lt")),
		events.NewBeerAvailabilityUpdated(beerId, commitId, beerName, customtypes.NewQuantity(70, "Lt"), orderId),
		events.NewQuantityNotFound(beerId, commitId, orderId, customtypes.NewQuantity(999, "Lt"), customtypes.NewQuantity(70, "Lt")),
		events.NewAvailabilityCompensated(beerId, commitId, beerName, customtypes.NewQuantity(100, "Lt"), orderId),
	})
	// A saga stream must be skipped by the rebuild.
	store.Seed(domain.SagaStreamName+"-"+orderId, []muflone.DomainEvent{
		events.NewOrderAllocationStarted(uuid.MustParse(orderId), commitId, nil),
	})
	freshReadModel := services.NewAvailabilityService()

	require.NoError(t, warehouses.RebuildReadModel(context.Background(), store, freshReadModel, slog.Default()))

	availability, err := freshReadModel.GetAvailability(context.Background(), beerId.Value.String())
	require.NoError(t, err)
	assert.Equal(t, 100, availability.Quantity.Value, "final compensated state")
}

type failingAvailabilityReadModel struct{ services.AvailabilityService }

func (f *failingAvailabilityReadModel) UpsertAvailability(context.Context, dtos.Availability) error {
	return assert.AnError
}

type failingWarehouseStore struct{ muflone.EventStore }

func (failingWarehouseStore) ListStreams(context.Context, string) ([]string, error) {
	return nil, assert.AnError
}

type readFailingWarehouseStore struct{ *muflone.InMemoryEventStore }

func (readFailingWarehouseStore) ReadStream(context.Context, string) ([]muflone.StoredEvent, error) {
	return nil, assert.AnError
}

func TestRebuildSurfacesFailures(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()
	store := muflone.NewInMemoryEventStore()
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	store.Seed(domain.StreamName+"-"+beerId.Value.String(), []muflone.DomainEvent{
		events.NewAvailabilityUpdatedDueToProductionOrder(beerId, uuid.New(),
			sharedkernel.BeerName{Value: "IPA"}, customtypes.NewQuantity(1, "Lt")),
	})

	assert.ErrorIs(t,
		warehouses.RebuildReadModel(ctx, store, &failingAvailabilityReadModel{}, logger),
		assert.AnError, "projection failure")
	assert.ErrorIs(t,
		warehouses.RebuildReadModel(ctx, failingWarehouseStore{}, services.NewAvailabilityService(), logger),
		assert.AnError, "stream listing failure")
	assert.ErrorIs(t,
		warehouses.RebuildReadModel(ctx, readFailingWarehouseStore{store}, services.NewAvailabilityService(), logger),
		assert.AnError, "stream read failure")
}
