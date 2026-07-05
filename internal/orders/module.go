// Package orders wires the orders bounded context into the DI container
// and subscribes its event handlers to the bus.
package orders

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/samber/do/v2"

	catalogqueries "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/infrastructure/acl"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/infrastructure/persistence"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/infrastructure/messaging"
)

func Register(injector do.Injector) {
	do.Provide(injector, func(i do.Injector) (domain.OrderRepository, error) {
		return persistence.NewMemoryOrderRepository(), nil
	})
	do.Provide(injector, func(i do.Injector) (ports.BeerCatalog, error) {
		return acl.NewCatalogAdapter(do.MustInvoke[*catalogqueries.GetBeerHandler](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.PlaceOrderHandler, error) {
		return commands.NewPlaceOrderHandler(
			do.MustInvoke[domain.OrderRepository](i),
			do.MustInvoke[ports.BeerCatalog](i),
			do.MustInvoke[sharedports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.SettleOrderHandler, error) {
		return commands.NewSettleOrderHandler(
			do.MustInvoke[domain.OrderRepository](i),
			do.MustInvoke[sharedports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.CancelOrderHandler, error) {
		return commands.NewCancelOrderHandler(
			do.MustInvoke[domain.OrderRepository](i),
			do.MustInvoke[sharedports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.GetOrderHandler, error) {
		return queries.NewGetOrderHandler(do.MustInvoke[domain.OrderRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.ListOrdersHandler, error) {
		return queries.NewListOrdersHandler(do.MustInvoke[domain.OrderRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*eventhandlers.Handlers, error) {
		return eventhandlers.NewHandlers(
			do.MustInvoke[*commands.SettleOrderHandler](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func SubscribeEventHandlers(injector do.Injector, bus *messaging.Bus) {
	handlers := do.MustInvoke[*eventhandlers.Handlers](injector)
	bus.Subscribe("orders.on_stock_reserved", handlers.StockReservedTopic(),
		func(msg *message.Message) error {
			return handlers.OnStockReserved(msg.Context(), msg.Payload)
		})
	bus.Subscribe("orders.on_stock_rejected", handlers.StockRejectedTopic(),
		func(msg *message.Message) error {
			return handlers.OnStockRejected(msg.Context(), msg.Payload)
		})
}
