// Package inventory wires the inventory bounded context into the DI
// container and subscribes its event handlers to the bus.
package inventory

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/infrastructure/persistence"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/infrastructure/messaging"
)

func Register(injector do.Injector) {
	do.Provide(injector, func(i do.Injector) (domain.StockRepository, error) {
		return persistence.NewMemoryStockRepository(), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.TrackStockItemHandler, error) {
		return commands.NewTrackStockItemHandler(do.MustInvoke[domain.StockRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.ReplenishStockHandler, error) {
		return commands.NewReplenishStockHandler(
			do.MustInvoke[domain.StockRepository](i),
			do.MustInvoke[ports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.ReserveOrderStockHandler, error) {
		return commands.NewReserveOrderStockHandler(
			do.MustInvoke[domain.StockRepository](i),
			do.MustInvoke[ports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.GetStockHandler, error) {
		return queries.NewGetStockHandler(do.MustInvoke[domain.StockRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.ListStockHandler, error) {
		return queries.NewListStockHandler(do.MustInvoke[domain.StockRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*eventhandlers.Handlers, error) {
		return eventhandlers.NewHandlers(
			do.MustInvoke[*commands.ReplenishStockHandler](i),
			do.MustInvoke[*commands.ReserveOrderStockHandler](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

// SubscribeEventHandlers attaches the inventory reactions to the bus. Kept
// separate from Register so wiring order (bus first) stays explicit in the
// composition root.
func SubscribeEventHandlers(injector do.Injector, bus *messaging.Bus) {
	handlers := do.MustInvoke[*eventhandlers.Handlers](injector)
	bus.Subscribe("inventory.on_batch_completed", handlers.BatchCompletedTopic(),
		func(msg *message.Message) error {
			return handlers.OnBatchCompleted(msg.Context(), msg.Payload)
		})
	bus.Subscribe("inventory.on_order_placed", handlers.OrderPlacedTopic(),
		func(msg *message.Message) error {
			return handlers.OnOrderPlaced(msg.Context(), msg.Payload)
		})
}
