// Package domain holds the Sales write model: the event-sourced SalesOrder
// aggregate — BrewUp.Sales.Domain.
package domain

import (
	"strings"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

// StreamName prefixes the event streams of this aggregate.
const StreamName = "SalesOrder"

// AllocationStatus is the order's position in the allocation saga.
type AllocationStatus string

const (
	AllocationPending  AllocationStatus = "pending"
	AllocationDone     AllocationStatus = "allocated"
	AllocationRejected AllocationStatus = "rejected"
)

var (
	ErrInvalidSalesOrder  = muflone.ErrInvalid("sales order needs a number, a customer, and an order date")
	ErrAllocationConflict = muflone.ErrInvalid("sales order allocation already settled differently")
)

// SalesOrder is event-sourced: the factory validates the invariants and
// raises SalesOrderCreated; apply() is the only place state is assigned,
// both when the event is first raised and when the repository replays the
// stream (the book's RaiseEvent / Apply pair).
type SalesOrder struct {
	muflone.AggregateRoot

	salesOrderId     sharedkernel.SalesOrderId
	salesOrderNumber sharedkernel.SalesOrderNumber
	orderDate        sharedkernel.OrderDate
	customerId       sharedkernel.CustomerId
	customerName     sharedkernel.CustomerName
	rows             []sharedkernel.SalesOrderRowDto
	allocation       AllocationStatus
	rejectionReason  string
}

// NewSalesOrder returns an empty aggregate ready for stream replay — the
// repository's factory (Muflone's ConstructAggregate).
func NewSalesOrder() *SalesOrder {
	order := &SalesOrder{}
	order.Bind(order)
	return order
}

// CreateSalesOrder is the aggregate factory: it checks the SalesOrder
// invariants and raises SalesOrderCreated.
func CreateSalesOrder(
	salesOrderId sharedkernel.SalesOrderId,
	commitId uuid.UUID,
	salesOrderNumber sharedkernel.SalesOrderNumber,
	orderDate sharedkernel.OrderDate,
	customerId sharedkernel.CustomerId,
	customerName sharedkernel.CustomerName,
	rows []sharedkernel.SalesOrderRowDto,
) (*SalesOrder, error) {
	if strings.TrimSpace(salesOrderNumber.Value) == "" ||
		strings.TrimSpace(customerName.Value) == "" ||
		orderDate.Value.IsZero() {
		return nil, ErrInvalidSalesOrder
	}
	order := NewSalesOrder()
	order.RaiseEvent(events.NewSalesOrderCreated(
		salesOrderId, commitId, salesOrderNumber, orderDate, customerId, customerName, rows,
	))
	return order, nil
}

// MarkAllocated settles the allocation saga's success. Idempotent: an
// order already allocated observes nothing new; a rejected order cannot
// flip to allocated.
func (s *SalesOrder) MarkAllocated(commitId uuid.UUID) (bool, error) {
	switch s.allocation {
	case AllocationDone:
		return false, nil
	case AllocationRejected:
		return false, ErrAllocationConflict
	}
	s.RaiseEvent(events.NewSalesOrderAllocated(s.salesOrderId, commitId))
	return true, nil
}

// MarkAllocationRejected settles the saga's failure (after compensation).
func (s *SalesOrder) MarkAllocationRejected(commitId uuid.UUID, reason string) (bool, error) {
	switch s.allocation {
	case AllocationRejected:
		return false, nil
	case AllocationDone:
		return false, ErrAllocationConflict
	}
	s.RaiseEvent(events.NewSalesOrderAllocationRejected(s.salesOrderId, commitId, reason))
	return true, nil
}

// Route dispatches events to the apply methods (RegisteredRoutes.Dispatch).
func (s *SalesOrder) Route(event muflone.DomainEvent) {
	switch e := event.(type) {
	case events.SalesOrderCreated:
		s.applySalesOrderCreated(e)
	case events.SalesOrderAllocated:
		s.allocation = AllocationDone
	case events.SalesOrderAllocationRejected:
		s.allocation = AllocationRejected
		s.rejectionReason = e.Reason
	}
}

func (s *SalesOrder) applySalesOrderCreated(event events.SalesOrderCreated) {
	s.SetID(event.SalesOrderId.Value)
	s.salesOrderId = event.SalesOrderId
	s.salesOrderNumber = event.SalesOrderNumber
	s.orderDate = event.OrderDate
	s.customerId = event.CustomerId
	s.customerName = event.CustomerName
	s.rows = event.Rows
	s.allocation = AllocationPending
}
