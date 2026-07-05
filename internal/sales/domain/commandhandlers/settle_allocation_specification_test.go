// Specification tests for settling the order from the allocation-saga
// outcome (book Ch. 12: the order status reacts to the saga).
package commandhandlers_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

func createdEvent(salesOrderId sharedkernel.SalesOrderId, commitId uuid.UUID) events.SalesOrderCreated {
	return events.NewSalesOrderCreated(
		salesOrderId, commitId,
		sharedkernel.SalesOrderNumber{Value: "20240315-1500"},
		sharedkernel.OrderDate{Value: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)},
		sharedkernel.CustomerId{Value: uuid.New()},
		sharedkernel.CustomerName{Value: "Muflone"},
		nil,
	)
}

func TestMarkSalesOrderAllocated(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.MarkSalesOrderAllocated]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{createdEvent(salesOrderId, correlationId)}
		},
		When: func() commands.MarkSalesOrderAllocated {
			return commands.NewMarkSalesOrderAllocated(salesOrderId, correlationId)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocated] {
			return commandhandlers.NewMarkSalesOrderAllocatedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{events.NewSalesOrderAllocated(salesOrderId, correlationId)}
		},
	}.Run(t)
}

func TestMarkSalesOrderAllocationRejected(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.MarkSalesOrderAllocationRejected]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{createdEvent(salesOrderId, correlationId)}
		},
		When: func() commands.MarkSalesOrderAllocationRejected {
			return commands.NewMarkSalesOrderAllocationRejected(salesOrderId, correlationId, "not enough stout")
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocationRejected] {
			return commandhandlers.NewMarkSalesOrderAllocationRejectedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewSalesOrderAllocationRejected(salesOrderId, correlationId, "not enough stout"),
			}
		},
	}.Run(t)
}

// TestSettlementIsIdempotent: a redelivered outcome commits nothing new.
func TestSettlementIsIdempotent(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.MarkSalesOrderAllocated]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				createdEvent(salesOrderId, correlationId),
				events.NewSalesOrderAllocated(salesOrderId, correlationId),
			}
		},
		When: func() commands.MarkSalesOrderAllocated {
			return commands.NewMarkSalesOrderAllocated(salesOrderId, correlationId)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocated] {
			return commandhandlers.NewMarkSalesOrderAllocatedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent { return nil },
	}.Run(t)

	muflone.CommandSpecification[commands.MarkSalesOrderAllocationRejected]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				createdEvent(salesOrderId, correlationId),
				events.NewSalesOrderAllocationRejected(salesOrderId, correlationId, "shortage"),
			}
		},
		When: func() commands.MarkSalesOrderAllocationRejected {
			return commands.NewMarkSalesOrderAllocationRejected(salesOrderId, correlationId, "shortage")
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocationRejected] {
			return commandhandlers.NewMarkSalesOrderAllocationRejectedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent { return nil },
	}.Run(t)
}

// TestConflictingSettlementFails: a saga cannot both complete and reject —
// a contradictory outcome is a hard error, nothing is committed.
func TestConflictingSettlementFails(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.MarkSalesOrderAllocated]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				createdEvent(salesOrderId, correlationId),
				events.NewSalesOrderAllocationRejected(salesOrderId, correlationId, "shortage"),
			}
		},
		When: func() commands.MarkSalesOrderAllocated {
			return commands.NewMarkSalesOrderAllocated(salesOrderId, correlationId)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocated] {
			return commandhandlers.NewMarkSalesOrderAllocatedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrAllocationConflict,
	}.Run(t)

	muflone.CommandSpecification[commands.MarkSalesOrderAllocationRejected]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				createdEvent(salesOrderId, correlationId),
				events.NewSalesOrderAllocated(salesOrderId, correlationId),
			}
		},
		When: func() commands.MarkSalesOrderAllocationRejected {
			return commands.NewMarkSalesOrderAllocationRejected(salesOrderId, correlationId, "late failure")
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocationRejected] {
			return commandhandlers.NewMarkSalesOrderAllocationRejectedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrAllocationConflict,
	}.Run(t)
}

// TestSettlementForUnknownOrderFails: the aggregate must exist.
func TestSettlementForUnknownOrderFails(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()

	muflone.CommandSpecification[commands.MarkSalesOrderAllocated]{
		StreamName: domain.StreamName,
		Given:      func() []muflone.DomainEvent { return nil },
		When: func() commands.MarkSalesOrderAllocated {
			return commands.NewMarkSalesOrderAllocated(salesOrderId, uuid.New())
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocated] {
			return commandhandlers.NewMarkSalesOrderAllocatedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: muflone.ErrAggregateNotFound,
	}.Run(t)

	muflone.CommandSpecification[commands.MarkSalesOrderAllocationRejected]{
		StreamName: domain.StreamName,
		Given:      func() []muflone.DomainEvent { return nil },
		When: func() commands.MarkSalesOrderAllocationRejected {
			return commands.NewMarkSalesOrderAllocationRejected(salesOrderId, uuid.New(), "x")
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.MarkSalesOrderAllocationRejected] {
			return commandhandlers.NewMarkSalesOrderAllocationRejectedCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: muflone.ErrAggregateNotFound,
	}.Run(t)
}
