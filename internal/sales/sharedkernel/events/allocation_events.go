package events

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
)

// SalesOrderAllocated: the warehouse holds stock for every row.
type SalesOrderAllocated struct {
	muflone.DomainEventBase
	SalesOrderId sharedkernel.SalesOrderId `json:"sales_order_id"`
}

func NewSalesOrderAllocated(salesOrderId sharedkernel.SalesOrderId, commitId uuid.UUID) SalesOrderAllocated {
	return SalesOrderAllocated{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId.Value, commitId),
		SalesOrderId:    salesOrderId,
	}
}

func (SalesOrderAllocated) MessageName() string { return "sales.sales_order_allocated" }

// SalesOrderAllocationRejected: the warehouse saga failed and compensated;
// nothing is held for this order.
type SalesOrderAllocationRejected struct {
	muflone.DomainEventBase
	SalesOrderId sharedkernel.SalesOrderId `json:"sales_order_id"`
	Reason       string                    `json:"reason"`
}

func NewSalesOrderAllocationRejected(
	salesOrderId sharedkernel.SalesOrderId,
	commitId uuid.UUID,
	reason string,
) SalesOrderAllocationRejected {
	return SalesOrderAllocationRejected{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId.Value, commitId),
		SalesOrderId:    salesOrderId,
		Reason:          reason,
	}
}

func (SalesOrderAllocationRejected) MessageName() string {
	return "sales.sales_order_allocation_rejected"
}
