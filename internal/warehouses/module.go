package warehouses

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/integration"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/integrationevents"
)

// Register wires the Warehouses module: event-sourced repository, command
// handlers, read-model projections, and the cross-context reactions.
func Register(injector do.Injector, bus *muflone.ServiceBus) {
	logger := do.MustInvoke[*slog.Logger](injector)

	// Write model.
	store := muflone.NewInMemoryEventStore()
	repository := muflone.NewEventStoreRepository(
		store, domain.NewAvailability, domain.StreamName, bus,
	)
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(repository, logger))
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(repository, logger))

	// Read model projections.
	queryService := services.NewAvailabilityService()
	projector := eventhandlers.NewAvailabilityProjector(queryService, logger)
	muflone.RegisterDomainEventHandler(bus, "warehouses.readmodel.availability_updated",
		projector.OnAvailabilityUpdatedDueToProductionOrder)
	muflone.RegisterDomainEventHandler(bus, "warehouses.readmodel.beer_availability_updated",
		projector.OnBeerAvailabilityUpdated)

	// Integration out: allocating stock notifies the other contexts.
	muflone.RegisterDomainEventHandler(bus, "warehouses.integration.beer_availability_updated",
		func(ctx context.Context, event events.BeerAvailabilityUpdated) error {
			return bus.PublishIntegrationEvent(ctx, integrationevents.NewBeerAvailabilityUpdated(
				event.BeerId.Value, event.CommitID(), event.BeerName.Value,
				event.Quantity.Value, event.Quantity.UnitOfMeasure,
			))
		})

	// Integration in: a sales order was created → allocate stock per row.
	salesOrderHandler := integration.NewSalesOrderCreatedHandler(bus, logger)
	bus.SubscribeIntegrationEvent("warehouses.on_sales_order_created",
		integration.SalesOrderCreatedTopic, salesOrderHandler.Handle)

	do.Provide(injector, func(i do.Injector) (*Facade, error) {
		return NewFacade(bus, queryService), nil
	})
}
