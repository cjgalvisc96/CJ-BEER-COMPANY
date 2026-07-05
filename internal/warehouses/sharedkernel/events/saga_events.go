package events

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
)

// The order-allocation saga is EVENT-SOURCED (book Ch. 12, "Event-sourced
// sagas"): every step's outcome is recorded as an event in the saga's own
// stream, so its state can always be rebuilt and the process resumed after
// a crash. The saga's aggregateId is the SalesOrderId.

// SagaRowData is one order row as the saga tracks it.
type SagaRowData struct {
	BeerId        string `json:"beer_id"`
	BeerName      string `json:"beer_name"`
	Quantity      int    `json:"quantity"`
	UnitOfMeasure string `json:"unit_of_measure"`
}

type OrderAllocationStarted struct {
	muflone.DomainEventBase
	SalesOrderId string        `json:"sales_order_id"`
	Rows         []SagaRowData `json:"rows"`
}

func NewOrderAllocationStarted(salesOrderId uuid.UUID, commitId uuid.UUID, rows []SagaRowData) OrderAllocationStarted {
	return OrderAllocationStarted{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
		Rows:            rows,
	}
}

func (OrderAllocationStarted) MessageName() string {
	return "warehouses.order_allocation_started"
}

type AllocationStepSucceeded struct {
	muflone.DomainEventBase
	SalesOrderId string `json:"sales_order_id"`
	BeerId       string `json:"beer_id"`
}

func NewAllocationStepSucceeded(salesOrderId uuid.UUID, commitId uuid.UUID, beerId string) AllocationStepSucceeded {
	return AllocationStepSucceeded{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
		BeerId:          beerId,
	}
}

func (AllocationStepSucceeded) MessageName() string {
	return "warehouses.allocation_step_succeeded"
}

type AllocationStepFailed struct {
	muflone.DomainEventBase
	SalesOrderId string `json:"sales_order_id"`
	BeerId       string `json:"beer_id"`
	Reason       string `json:"reason"`
}

func NewAllocationStepFailed(salesOrderId uuid.UUID, commitId uuid.UUID, beerId, reason string) AllocationStepFailed {
	return AllocationStepFailed{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
		BeerId:          beerId,
		Reason:          reason,
	}
}

func (AllocationStepFailed) MessageName() string {
	return "warehouses.allocation_step_failed"
}

type AllocationStepCompensated struct {
	muflone.DomainEventBase
	SalesOrderId string `json:"sales_order_id"`
	BeerId       string `json:"beer_id"`
}

func NewAllocationStepCompensated(salesOrderId uuid.UUID, commitId uuid.UUID, beerId string) AllocationStepCompensated {
	return AllocationStepCompensated{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
		BeerId:          beerId,
	}
}

func (AllocationStepCompensated) MessageName() string {
	return "warehouses.allocation_step_compensated"
}

type OrderAllocationCompleted struct {
	muflone.DomainEventBase
	SalesOrderId string `json:"sales_order_id"`
}

func NewOrderAllocationCompleted(salesOrderId uuid.UUID, commitId uuid.UUID) OrderAllocationCompleted {
	return OrderAllocationCompleted{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
	}
}

func (OrderAllocationCompleted) MessageName() string {
	return "warehouses.order_allocation_completed"
}

type OrderAllocationRejected struct {
	muflone.DomainEventBase
	SalesOrderId string `json:"sales_order_id"`
	Reason       string `json:"reason"`
}

func NewOrderAllocationRejected(salesOrderId uuid.UUID, commitId uuid.UUID, reason string) OrderAllocationRejected {
	return OrderAllocationRejected{
		DomainEventBase: muflone.NewDomainEventBase(salesOrderId, commitId),
		SalesOrderId:    salesOrderId.String(),
		Reason:          reason,
	}
}

func (OrderAllocationRejected) MessageName() string {
	return "warehouses.order_allocation_rejected"
}
