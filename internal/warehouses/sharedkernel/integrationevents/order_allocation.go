package integrationevents

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
)

const (
	OrderAllocationCompletedName = "warehouses.order_allocation_completed"
	OrderAllocationRejectedName  = "warehouses.order_allocation_rejected"
)

// OrderAllocationCompleted tells the other contexts (Sales) that every row
// of the order is allocated.
type OrderAllocationCompleted struct {
	muflone.IntegrationEventBase
	SalesOrderId string `json:"sales_order_id"`
}

func NewOrderAllocationCompleted(salesOrderId string, commitId uuid.UUID) OrderAllocationCompleted {
	return OrderAllocationCompleted{
		IntegrationEventBase: muflone.NewIntegrationEventBase(commitId),
		SalesOrderId:         salesOrderId,
	}
}

func (OrderAllocationCompleted) MessageName() string { return OrderAllocationCompletedName }

// OrderAllocationRejected tells the other contexts the saga failed and was
// compensated: nothing is held for this order anymore.
type OrderAllocationRejected struct {
	muflone.IntegrationEventBase
	SalesOrderId string `json:"sales_order_id"`
	Reason       string `json:"reason"`
}

func NewOrderAllocationRejected(salesOrderId string, commitId uuid.UUID, reason string) OrderAllocationRejected {
	return OrderAllocationRejected{
		IntegrationEventBase: muflone.NewIntegrationEventBase(commitId),
		SalesOrderId:         salesOrderId,
		Reason:               reason,
	}
}

func (OrderAllocationRejected) MessageName() string { return OrderAllocationRejectedName }
