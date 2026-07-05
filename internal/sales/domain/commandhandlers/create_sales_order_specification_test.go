// Specification tests for the SalesOrder aggregate — the Go rendition of
// the book's Example 1 ("Creating a Sales Order"): Given past events, When
// a command is handled, Expect the committed events.
package commandhandlers_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

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

// TestCreateSalesOrderIsIdempotent: retrying a creation with the same
// client-supplied id acknowledges without duplicating (safe retries).
func TestCreateSalesOrderIsIdempotent(t *testing.T) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.CreateSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{createdEvent(salesOrderId, correlationId)}
		},
		When: func() commands.CreateSalesOrder {
			return commands.NewCreateSalesOrder(
				salesOrderId, correlationId,
				sharedkernel.SalesOrderNumber{Value: "20240315-1500"},
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
		Expect: func() []muflone.DomainEvent { return nil },
	}.Run(t)
}

// failingSalesRepository drives the infrastructure-error branch of the
// idempotency check.
type failingSalesRepository struct{}

func (failingSalesRepository) GetByID(context.Context, uuid.UUID) (*domain.SalesOrder, error) {
	return nil, assert.AnError
}

func (failingSalesRepository) Save(context.Context, *domain.SalesOrder, uuid.UUID) error {
	return assert.AnError
}

func TestCreateSalesOrderSurfacesStoreFailures(t *testing.T) {
	handler := commandhandlers.NewCreateSalesOrderCommandHandler(failingSalesRepository{}, slog.Default())

	err := handler.Handle(context.Background(), commands.NewCreateSalesOrder(
		sharedkernel.NewSalesOrderId(), uuid.New(),
		sharedkernel.SalesOrderNumber{Value: "x"},
		sharedkernel.OrderDate{Value: time.Now().UTC()},
		sharedkernel.CustomerId{Value: uuid.New()},
		sharedkernel.CustomerName{Value: "y"}, nil,
	))

	assert.ErrorIs(t, err, assert.AnError)
}
