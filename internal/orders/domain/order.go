// Package domain models the orders bounded context: customer purchases and
// their lifecycle (pending → confirmed/rejected, confirmed → cancelled).
package domain

import (
	"strings"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type OrderID struct {
	shared.EntityID
}

func NewOrderID() OrderID {
	return OrderID{EntityID: shared.NewEntityID()}
}

func ParseOrderID(raw string) (OrderID, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return OrderID{}, err
	}
	return OrderID{EntityID: id}, nil
}

type OrderStatus string

const (
	// OrderStatusPending means the order was placed and is waiting for the
	// inventory context to reserve stock.
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order is the aggregate root. Lines are immutable once the order is
// placed; state changes only through the lifecycle methods below.
type Order struct {
	shared.AggregateRoot

	id           OrderID
	customerName string
	lines        []OrderLine
	status       OrderStatus
	rejectReason string
}

// PlaceOrder creates a pending order and records OrderPlaced. Prices are
// captured at placement time: later catalog price changes must not affect
// an order already placed.
func PlaceOrder(customerName string, lines []OrderLine) (*Order, error) {
	customerName = strings.TrimSpace(customerName)
	if customerName == "" {
		return nil, ErrEmptyCustomerName
	}
	if len(lines) == 0 {
		return nil, ErrEmptyOrder
	}
	order := &Order{
		id:           NewOrderID(),
		customerName: customerName,
		lines:        lines,
		status:       OrderStatusPending,
	}
	total, err := order.Total()
	if err != nil {
		return nil, err
	}
	eventLines := make([]OrderPlacedLine, 0, len(lines))
	for _, line := range lines {
		eventLines = append(eventLines, OrderPlacedLine{
			BeerID: line.BeerID().String(),
			Units:  line.Units(),
		})
	}
	order.RecordEvent(OrderPlaced{
		BaseEvent:    shared.NewBaseEvent(),
		OrderID:      order.id.String(),
		CustomerName: customerName,
		Lines:        eventLines,
		TotalCents:   total.Cents(),
		Currency:     total.Currency(),
	})
	return order, nil
}

// RehydrateOrder rebuilds an Order from persisted state.
func RehydrateOrder(
	id OrderID,
	customerName string,
	lines []OrderLine,
	status OrderStatus,
	rejectReason string,
) *Order {
	return &Order{
		id:           id,
		customerName: customerName,
		lines:        lines,
		status:       status,
		rejectReason: rejectReason,
	}
}

func (o *Order) ID() OrderID          { return o.id }
func (o *Order) CustomerName() string { return o.customerName }
func (o *Order) Status() OrderStatus  { return o.status }
func (o *Order) RejectReason() string { return o.rejectReason }

// Lines returns a defensive copy so callers cannot mutate the aggregate.
func (o *Order) Lines() []OrderLine {
	lines := make([]OrderLine, len(o.lines))
	copy(lines, o.lines)
	return lines
}

// Total sums the line subtotals.
func (o *Order) Total() (shared.Money, error) {
	total := o.lines[0].Subtotal()
	for _, line := range o.lines[1:] {
		summed, err := total.Add(line.Subtotal())
		if err != nil {
			return shared.Money{}, err
		}
		total = summed
	}
	return total, nil
}

// Confirm transitions pending → confirmed once stock has been reserved.
func (o *Order) Confirm() error {
	if o.status != OrderStatusPending {
		return ErrOrderNotPending
	}
	o.status = OrderStatusConfirmed
	o.RecordEvent(OrderConfirmed{
		BaseEvent: shared.NewBaseEvent(),
		OrderID:   o.id.String(),
	})
	return nil
}

// Reject transitions pending → rejected when stock could not be reserved.
func (o *Order) Reject(reason string) error {
	if o.status != OrderStatusPending {
		return ErrOrderNotPending
	}
	o.status = OrderStatusRejected
	o.rejectReason = reason
	o.RecordEvent(OrderRejected{
		BaseEvent: shared.NewBaseEvent(),
		OrderID:   o.id.String(),
		Reason:    reason,
	})
	return nil
}

// Cancel lets a customer back out of an order that is not yet (or just)
// confirmed. Rejected orders are already terminal.
func (o *Order) Cancel() error {
	switch o.status {
	case OrderStatusPending, OrderStatusConfirmed:
		o.status = OrderStatusCancelled
		o.RecordEvent(OrderCancelled{
			BaseEvent: shared.NewBaseEvent(),
			OrderID:   o.id.String(),
		})
		return nil
	default:
		return ErrOrderNotCancellable
	}
}
