package commands

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
)

// MarkSalesOrderAllocated settles the order after the warehouse saga
// completed: every row has stock held for it.
type MarkSalesOrderAllocated struct {
	muflone.CommandBase
	SalesOrderId sharedkernel.SalesOrderId `json:"sales_order_id"`
}

func NewMarkSalesOrderAllocated(salesOrderId sharedkernel.SalesOrderId, commitId uuid.UUID) MarkSalesOrderAllocated {
	return MarkSalesOrderAllocated{
		CommandBase:  muflone.NewCommandBase(salesOrderId.Value, commitId),
		SalesOrderId: salesOrderId,
	}
}

func (MarkSalesOrderAllocated) MessageName() string {
	return "sales.mark_sales_order_allocated"
}

// MarkSalesOrderAllocationRejected settles the order after the warehouse
// saga failed and compensated (book Ch. 12: the order status changes in
// reaction to the failure).
type MarkSalesOrderAllocationRejected struct {
	muflone.CommandBase
	SalesOrderId sharedkernel.SalesOrderId `json:"sales_order_id"`
	Reason       string                    `json:"reason"`
}

func NewMarkSalesOrderAllocationRejected(
	salesOrderId sharedkernel.SalesOrderId,
	commitId uuid.UUID,
	reason string,
) MarkSalesOrderAllocationRejected {
	return MarkSalesOrderAllocationRejected{
		CommandBase:  muflone.NewCommandBase(salesOrderId.Value, commitId),
		SalesOrderId: salesOrderId,
		Reason:       reason,
	}
}

func (MarkSalesOrderAllocationRejected) MessageName() string {
	return "sales.mark_sales_order_allocation_rejected"
}
