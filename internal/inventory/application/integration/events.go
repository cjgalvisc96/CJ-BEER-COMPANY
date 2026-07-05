// Package integration defines the application-level integration events the
// inventory context publishes as the outcome of the order-reservation
// process. They are not aggregate events: "all lines of order X reserved"
// is a fact about a process spanning several StockItem aggregates.
package integration

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

const (
	OrderStockReservedTopic = "inventory.order_stock_reserved"
	OrderStockRejectedTopic = "inventory.order_stock_rejected"
)

type OrderStockReserved struct {
	shared.BaseEvent
	OrderID string `json:"order_id"`
}

func (OrderStockReserved) EventName() string { return OrderStockReservedTopic }

type OrderStockRejected struct {
	shared.BaseEvent
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

func (OrderStockRejected) EventName() string { return OrderStockRejectedTopic }
