package sales

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/integration"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/integrationevents"
)

// NewEventRegistry knows every event this module stores — also used by
// the projection-rebuild command (cmd/rebuild).
func NewEventRegistry() *muflone.EventRegistry {
	registry := muflone.NewEventRegistry()
	muflone.RegisterEvent[events.SalesOrderCreated](registry)
	muflone.RegisterEvent[events.SalesOrderAllocated](registry)
	muflone.RegisterEvent[events.SalesOrderAllocationRejected](registry)
	return registry
}

// salesOrderReadModel is what the module needs from a read-model adapter:
// the projection writers plus the facade queries.
type salesOrderReadModel interface {
	CreateSalesOrder(ctx context.Context, order dtos.SalesOrder) error
	UpdateAllocationStatus(ctx context.Context, salesOrderId, status, reason string) error
	SalesOrderQueries
}

// Register wires the Sales module: event registry, event-sourced
// repository (durable when DB_URL is configured, in-memory otherwise),
// command handlers, read-model projections, the integration publisher,
// and the reactions settling the order from the allocation-saga outcome.
func Register(injector do.Injector, bus *muflone.ServiceBus) {
	logger := do.MustInvoke[*slog.Logger](injector)
	cfg := do.MustInvoke[config.Config](injector)

	// The registry rehydrates this module's events from the store; new
	// event versions register their upcasters here (book Ch. 11).
	registry := NewEventRegistry()

	// Write + read model adapters: each context owns its data.
	var store muflone.EventStore = muflone.NewInMemoryEventStore()
	var readModel salesOrderReadModel = services.NewSalesOrderService()
	if cfg.DBURL != "" {
		db := do.MustInvoke[*sql.DB](injector)
		store = muflone.NewPostgresEventStore(db, registry)
		readModel = services.NewPostgresSalesOrderService(db)
	}

	repository := muflone.NewEventStoreRepository(
		store, domain.NewSalesOrder, domain.StreamName, bus,
	)
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewCreateSalesOrderCommandHandler(repository, logger))
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewMarkSalesOrderAllocatedCommandHandler(repository, logger))
	muflone.RegisterCommandHandler(bus,
		commandhandlers.NewMarkSalesOrderAllocationRejectedCommandHandler(repository, logger))

	// Read model: projections subscribe to the module's domain events.
	projection := eventhandlers.NewSalesOrderCreatedEventHandler(readModel, logger)
	muflone.RegisterDomainEventHandler(bus, "sales.readmodel.sales_order_created",
		projection.Handle)
	statusProjector := eventhandlers.NewAllocationStatusProjector(readModel, logger)
	muflone.RegisterDomainEventHandler(bus, "sales.readmodel.sales_order_allocated",
		statusProjector.OnSalesOrderAllocated)
	muflone.RegisterDomainEventHandler(bus, "sales.readmodel.sales_order_allocation_rejected",
		statusProjector.OnSalesOrderAllocationRejected)

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

	// The warehouse's allocation saga answers with its outcome: settle the
	// order accordingly (book Ch. 12, Figure 12.3).
	outcomes := integration.NewAllocationOutcomeHandler(bus, logger)
	bus.SubscribeIntegrationEvent("sales.on_order_allocation_completed",
		integration.OrderAllocationCompletedTopic, outcomes.OnAllocationCompleted)
	bus.SubscribeIntegrationEvent("sales.on_order_allocation_rejected",
		integration.OrderAllocationRejectedTopic, outcomes.OnAllocationRejected)

	do.Provide(injector, func(i do.Injector) (*Facade, error) {
		return NewFacade(bus, readModel), nil
	})
}
