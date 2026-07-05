package warehouses

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sagas"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

// NewEventRegistry knows every event this module stores — also used by
// the projection-rebuild command (cmd/rebuild).
func NewEventRegistry() *muflone.EventRegistry {
	registry := muflone.NewEventRegistry()
	muflone.RegisterEvent[events.AvailabilityUpdatedDueToProductionOrder](registry)
	muflone.RegisterEvent[events.BeerAvailabilityUpdated](registry)
	muflone.RegisterEvent[events.QuantityNotFound](registry)
	muflone.RegisterEvent[events.AvailabilityCompensated](registry)
	muflone.RegisterEvent[events.OrderAllocationStarted](registry)
	muflone.RegisterEvent[events.AllocationStepSucceeded](registry)
	muflone.RegisterEvent[events.AllocationStepFailed](registry)
	muflone.RegisterEvent[events.AllocationStepCompensated](registry)
	muflone.RegisterEvent[events.OrderAllocationCompleted](registry)
	muflone.RegisterEvent[events.OrderAllocationRejected](registry)
	return registry
}

// availabilityReadModel is what the module needs from a read-model
// adapter: the projection writer plus the facade queries.
type availabilityReadModel interface {
	UpsertAvailability(ctx context.Context, availability dtos.Availability) error
	AvailabilityQueries
}

// Register wires the Warehouses module: event registry, event-sourced
// repositories (durable when DB_URL is configured, in-memory otherwise),
// command handlers, read-model projections, and the order-allocation saga.
func Register(injector do.Injector, bus *muflone.ServiceBus) {
	logger := do.MustInvoke[*slog.Logger](injector)
	cfg := do.MustInvoke[config.Config](injector)

	registry := NewEventRegistry()

	var store muflone.EventStore = muflone.NewInMemoryEventStore()
	var readModel availabilityReadModel = services.NewAvailabilityService()
	if cfg.DBURL != "" {
		db := do.MustInvoke[*sql.DB](injector)
		store = muflone.NewPostgresEventStore(db, registry)
		readModel = services.NewPostgresAvailabilityService(db)
	}

	repository := muflone.NewEventStoreRepository(
		store, domain.NewAvailability, domain.StreamName, bus,
	)
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(repository, logger))
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(repository, logger))
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewCompensateAvailabilityDueToFailedAllocationCommandHandler(repository, logger))

	// Read model projections: every event carrying the new cumulative
	// quantity is an upsert. QuantityNotFound projects nothing.
	projector := eventhandlers.NewAvailabilityProjector(readModel, logger)
	muflone.RegisterDomainEventHandler(bus, "warehouses.readmodel.availability_updated",
		projector.OnAvailabilityUpdatedDueToProductionOrder)
	muflone.RegisterDomainEventHandler(bus, "warehouses.readmodel.beer_availability_updated",
		projector.OnBeerAvailabilityUpdated)
	muflone.RegisterDomainEventHandler(bus, "warehouses.readmodel.availability_compensated",
		projector.OnAvailabilityCompensated)

	// The order-allocation saga (book Ch. 12): event-sourced in the same
	// event store, triggered by the Sales integration event, advanced by
	// this module's own step events.
	sagaRepository := muflone.NewEventStoreRepository(
		store, domain.NewOrderAllocationSaga, domain.SagaStreamName, bus,
	)
	saga := sagas.NewOrderAllocationSaga(sagaRepository, store, bus, logger)
	bus.SubscribeIntegrationEvent("warehouses.saga.on_sales_order_created",
		sagas.SalesOrderCreatedTopic, saga.OnSalesOrderCreated)
	muflone.RegisterDomainEventHandler(bus, "warehouses.saga.on_beer_availability_updated",
		saga.OnBeerAvailabilityUpdated)
	muflone.RegisterDomainEventHandler(bus, "warehouses.saga.on_quantity_not_found",
		saga.OnQuantityNotFound)
	muflone.RegisterDomainEventHandler(bus, "warehouses.saga.on_availability_compensated",
		saga.OnAvailabilityCompensated)
	// The composition root resumes in-flight sagas once the bus runs, and
	// drives the step-timeout watchdog (durable execution, ADR-0008).
	do.ProvideValue(injector, saga)

	do.Provide(injector, func(i do.Injector) (*Facade, error) {
		return NewFacade(bus, readModel), nil
	})
}
