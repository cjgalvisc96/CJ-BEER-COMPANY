// Package commands holds the Sales module's commands (imperative form) —
// BrewUp.Sales.SharedKernel/Commands.
package commands

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
)

// CreateSalesOrder mirrors the book's command one to one:
// CreateSalesOrder(aggregateId, commitId, salesOrderNumber, orderDate,
// customerId, customerName, rows) : Command(aggregateId, commitId).
type CreateSalesOrder struct {
	muflone.CommandBase
	SalesOrderId     sharedkernel.SalesOrderId       `json:"sales_order_id"`
	SalesOrderNumber sharedkernel.SalesOrderNumber   `json:"sales_order_number"`
	OrderDate        sharedkernel.OrderDate          `json:"order_date"`
	CustomerId       sharedkernel.CustomerId         `json:"customer_id"`
	CustomerName     sharedkernel.CustomerName       `json:"customer_name"`
	Rows             []sharedkernel.SalesOrderRowDto `json:"rows"`
}

func NewCreateSalesOrder(
	salesOrderId sharedkernel.SalesOrderId,
	commitId uuid.UUID,
	salesOrderNumber sharedkernel.SalesOrderNumber,
	orderDate sharedkernel.OrderDate,
	customerId sharedkernel.CustomerId,
	customerName sharedkernel.CustomerName,
	rows []sharedkernel.SalesOrderRowDto,
) CreateSalesOrder {
	return CreateSalesOrder{
		CommandBase:      muflone.NewCommandBase(salesOrderId.Value, commitId),
		SalesOrderId:     salesOrderId,
		SalesOrderNumber: salesOrderNumber,
		OrderDate:        orderDate,
		CustomerId:       customerId,
		CustomerName:     customerName,
		Rows:             rows,
	}
}

func (CreateSalesOrder) MessageName() string { return "sales.create_sales_order" }
