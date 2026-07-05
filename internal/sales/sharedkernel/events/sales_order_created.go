// Package events holds the Sales module's domain events (past tense) —
// BrewUp.Sales.SharedKernel/Events. Domain events stay inside the module.
package events

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
)

// SalesOrderCreated mirrors the book's event:
// SalesOrderCreated(aggregateId, commitId, salesOrderNumber, orderDate,
// customerId, customerName, rows) : DomainEvent(aggregateId, commitId).
type SalesOrderCreated struct {
	muflone.DomainEventBase
	SalesOrderId     sharedkernel.SalesOrderId       `json:"sales_order_id"`
	SalesOrderNumber sharedkernel.SalesOrderNumber   `json:"sales_order_number"`
	OrderDate        sharedkernel.OrderDate          `json:"order_date"`
	CustomerId       sharedkernel.CustomerId         `json:"customer_id"`
	CustomerName     sharedkernel.CustomerName       `json:"customer_name"`
	Rows             []sharedkernel.SalesOrderRowDto `json:"rows"`
}

func NewSalesOrderCreated(
	salesOrderId sharedkernel.SalesOrderId,
	commitId uuid.UUID,
	salesOrderNumber sharedkernel.SalesOrderNumber,
	orderDate sharedkernel.OrderDate,
	customerId sharedkernel.CustomerId,
	customerName sharedkernel.CustomerName,
	rows []sharedkernel.SalesOrderRowDto,
) SalesOrderCreated {
	return SalesOrderCreated{
		DomainEventBase:  muflone.NewDomainEventBase(salesOrderId.Value, commitId),
		SalesOrderId:     salesOrderId,
		SalesOrderNumber: salesOrderNumber,
		OrderDate:        orderDate,
		CustomerId:       customerId,
		CustomerName:     customerName,
		Rows:             rows,
	}
}

func (SalesOrderCreated) MessageName() string { return "sales.sales_order_created" }
