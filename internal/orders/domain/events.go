package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

const (
	OrderPlacedTopic    = "orders.order_placed"
	OrderConfirmedTopic = "orders.order_confirmed"
	OrderRejectedTopic  = "orders.order_rejected"
	OrderCancelledTopic = "orders.order_cancelled"
)

type OrderPlacedLine struct {
	BeerID string `json:"beer_id"`
	Units  int    `json:"units"`
}

type OrderPlaced struct {
	shared.BaseEvent
	OrderID      string            `json:"order_id"`
	CustomerName string            `json:"customer_name"`
	Lines        []OrderPlacedLine `json:"lines"`
	TotalCents   int64             `json:"total_cents"`
	Currency     string            `json:"currency"`
}

func (OrderPlaced) EventName() string { return OrderPlacedTopic }

type OrderConfirmed struct {
	shared.BaseEvent
	OrderID string `json:"order_id"`
}

func (OrderConfirmed) EventName() string { return OrderConfirmedTopic }

type OrderRejected struct {
	shared.BaseEvent
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

func (OrderRejected) EventName() string { return OrderRejectedTopic }

type OrderCancelled struct {
	shared.BaseEvent
	OrderID string `json:"order_id"`
}

func (OrderCancelled) EventName() string { return OrderCancelledTopic }
