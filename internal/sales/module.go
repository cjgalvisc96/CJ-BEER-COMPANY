package sales

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/integrationevents"
)

// Register wires the Sales module: event-sourced repository, command
// handler, read-model projections, and the integration publisher. It is
// the module's composition root.
func Register(injector do.Injector, bus *muflone.ServiceBus) {
	logger := do.MustInvoke[*slog.Logger](injector)

	// Write model: one event store per module — each context owns its data.
	store := muflone.NewInMemoryEventStore()
	repository := muflone.NewEventStoreRepository(
		store, domain.NewSalesOrder, domain.StreamName, bus,
	)
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewCreateSalesOrderCommandHandler(repository, logger))

	// Read model: projections subscribe to the module's domain events.
	queryService := services.NewSalesOrderService()
	projection := eventhandlers.NewSalesOrderCreatedEventHandler(queryService, logger)
	muflone.RegisterDomainEventHandler(bus, "sales.readmodel.sales_order_created",
		projection.Handle)

	// Integration: republish the fact for other bounded contexts as a
	// dedicated integration event (never the domain event itself).
	muflone.RegisterDomainEventHandler(bus, "sales.integration.sales_order_created",
		func(ctx context.Context, event events.SalesOrderCreated) error {
			rows := make([]integrationevents.SalesOrderCreatedRow, 0, len(event.Rows))
			for _, row := range event.Rows {
				rows = append(rows, integrationevents.SalesOrderCreatedRow{
					BeerId:        row.BeerId.Value.String(),
					BeerName:      row.BeerName.Value,
					Quantity:      row.Quantity.Value,
					UnitOfMeasure: row.Quantity.UnitOfMeasure,
				})
			}
			return bus.PublishIntegrationEvent(ctx, integrationevents.NewSalesOrderCreated(
				event.SalesOrderId.Value, event.CommitID(), rows,
			))
		})

	// The warehouse answers with BeerAvailabilityUpdated (integration):
	// Sales is notified that stock was allocated for its order.
	bus.SubscribeIntegrationEvent("sales.on_beer_availability_updated",
		"warehouses.beer_availability_updated",
		func(ctx context.Context, payload []byte) error {
			logger.Info("sales.stock_allocation_notified", slog.String("payload", string(payload)))
			return nil
		})

	do.Provide(injector, func(i do.Injector) (*Facade, error) {
		return NewFacade(bus, queryService), nil
	})
}
