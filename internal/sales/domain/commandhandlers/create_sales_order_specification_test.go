// Specification tests for the SalesOrder aggregate — the Go rendition of
// the book's Example 1 ("Creating a Sales Order"): Given past events, When
// a command is handled, Expect the committed events.
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

func newSalesOrderRepository(store muflone.EventStore) muflone.Repository[*domain.SalesOrder] {
	return muflone.NewEventStoreRepository(store, domain.NewSalesOrder, domain.StreamName, nil)
}

// TestCreateSalesOrderSuccessfully mirrors CreateSalesOrderSuccessfully:
// no prior events, a CreateSalesOrder command, and exactly one
// SalesOrderCreated with the same data is expected.
func TestCreateSalesOrderSuccessfully(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	salesOrderNumber := sharedkernel.SalesOrderNumber{Value: "20240315-1500"}
	orderDate := sharedkernel.OrderDate{Value: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)}
	correlationId := uuid.New()
	customerId := sharedkernel.CustomerId{Value: uuid.New()}
	customerName := sharedkernel.CustomerName{Value: "Muflone"}
	rows := []sharedkernel.SalesOrderRowDto{}

	muflone.CommandSpecification[commands.CreateSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return nil
		},
		When: func() commands.CreateSalesOrder {
			return commands.NewCreateSalesOrder(
				salesOrderId, correlationId, salesOrderNumber, orderDate,
				customerId, customerName, rows,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.CreateSalesOrder] {
			return commandhandlers.NewCreateSalesOrderCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewSalesOrderCreated(
					salesOrderId, correlationId, salesOrderNumber, orderDate,
					customerId, customerName, rows,
				),
			}
		},
	}.Run(t)
}

// TestCreateSalesOrderWithoutNumberFails: violated invariants commit
// nothing and surface a domain error.
func TestCreateSalesOrderWithoutNumberFails(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()

	muflone.CommandSpecification[commands.CreateSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return nil
		},
		When: func() commands.CreateSalesOrder {
			return commands.NewCreateSalesOrder(
				salesOrderId, uuid.New(),
				sharedkernel.SalesOrderNumber{Value: "   "},
				sharedkernel.OrderDate{Value: time.Now().UTC()},
				sharedkernel.CustomerId{Value: uuid.New()},
				sharedkernel.CustomerName{Value: "Muflone"},
				nil,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.CreateSalesOrder] {
			return commandhandlers.NewCreateSalesOrderCommandHandler(
				newSalesOrderRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrInvalidSalesOrder,
	}.Run(t)
}
