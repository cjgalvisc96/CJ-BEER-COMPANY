package domain

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

// SagaStreamName prefixes the saga's event streams (one per sales order).
const SagaStreamName = "OrderAllocationSaga"

// SagaStatus is the saga's lifecycle. Allocation runs one row at a time
// (each step is a local transaction on one Availability aggregate); a
// failed step flips the saga into compensating, which undoes every
// already-allocated row before rejecting the order — the book's backward
// recovery (Ch. 12).
type SagaStatus string

const (
	SagaRunning      SagaStatus = "running"
	SagaCompensating SagaStatus = "compensating"
	SagaCompleted    SagaStatus = "completed"
	SagaRejected     SagaStatus = "rejected"
)

type sagaRowState string

const (
	rowPending     sagaRowState = "pending"
	rowAllocated   sagaRowState = "allocated"
	rowFailed      sagaRowState = "failed"
	rowCompensated sagaRowState = "compensated"
)

type sagaRow struct {
	data  events.SagaRowData
	state sagaRowState
}

// OrderAllocationSaga is an event-sourced saga (book Ch. 12): its state is
// derived from its own event stream, so a crashed process can rebuild it
// and resume. All Record* methods are IDEMPOTENT — redelivered messages
// re-observe facts the saga already knows and raise nothing new; they
// report whether anything changed so the runner only acts on transitions.
type OrderAllocationSaga struct {
	muflone.AggregateRoot

	salesOrderId string
	rows         []sagaRow
	status       SagaStatus
	reason       string
}

// NewOrderAllocationSaga returns an empty saga for stream replay.
func NewOrderAllocationSaga() *OrderAllocationSaga {
	saga := &OrderAllocationSaga{}
	saga.Bind(saga)
	return saga
}

// StartOrderAllocation begins the saga for one sales order. An order with
// no rows completes immediately.
func StartOrderAllocation(salesOrderId uuid.UUID, commitId uuid.UUID, rows []events.SagaRowData) *OrderAllocationSaga {
	saga := NewOrderAllocationSaga()
	saga.RaiseEvent(events.NewOrderAllocationStarted(salesOrderId, commitId, rows))
	if len(rows) == 0 {
		saga.RaiseEvent(events.NewOrderAllocationCompleted(salesOrderId, commitId))
	}
	return saga
}

func (s *OrderAllocationSaga) SalesOrderId() string { return s.salesOrderId }
func (s *OrderAllocationSaga) Status() SagaStatus   { return s.status }
func (s *OrderAllocationSaga) Reason() string       { return s.reason }

// NextPendingRow returns the next allocation step while the saga runs. A
// running saga always has one: completion is raised in the same transition
// that allocates the last row.
func (s *OrderAllocationSaga) NextPendingRow() (events.SagaRowData, bool) {
	if s.status == SagaRunning {
		for _, row := range s.rows {
			if row.state == rowPending {
				return row.data, true
			}
		}
	}
	return events.SagaRowData{}, false
}

// RowsToCompensate lists the rows still holding stock while compensating.
func (s *OrderAllocationSaga) RowsToCompensate() []events.SagaRowData {
	if s.status != SagaCompensating {
		return nil
	}
	var rows []events.SagaRowData
	for _, row := range s.rows {
		if row.state == rowAllocated {
			rows = append(rows, row.data)
		}
	}
	return rows
}

// RecordAllocationSucceeded observes a successful step. When every row is
// allocated the saga completes.
func (s *OrderAllocationSaga) RecordAllocationSucceeded(commitId uuid.UUID, beerId string) bool {
	if s.status != SagaRunning || !s.rowInState(beerId, rowPending) {
		return false
	}
	orderId := uuid.MustParse(s.salesOrderId)
	s.RaiseEvent(events.NewAllocationStepSucceeded(orderId, commitId, beerId))
	if s.allRowsIn(rowAllocated) {
		s.RaiseEvent(events.NewOrderAllocationCompleted(orderId, commitId))
	}
	return true
}

// RecordQuantityNotFound observes a failed step: the saga flips to
// compensating; with nothing to undo it rejects immediately.
func (s *OrderAllocationSaga) RecordQuantityNotFound(commitId uuid.UUID, beerId, reason string) bool {
	if s.status != SagaRunning || !s.rowInState(beerId, rowPending) {
		return false
	}
	orderId := uuid.MustParse(s.salesOrderId)
	s.RaiseEvent(events.NewAllocationStepFailed(orderId, commitId, beerId, reason))
	if len(s.RowsToCompensate()) == 0 {
		s.RaiseEvent(events.NewOrderAllocationRejected(orderId, commitId, reason))
	}
	return true
}

// RecordCompensated observes one undone row. When nothing is left holding
// stock the saga rejects the order.
func (s *OrderAllocationSaga) RecordCompensated(commitId uuid.UUID, beerId string) bool {
	if s.status != SagaCompensating || !s.rowInState(beerId, rowAllocated) {
		return false
	}
	orderId := uuid.MustParse(s.salesOrderId)
	s.RaiseEvent(events.NewAllocationStepCompensated(orderId, commitId, beerId))
	if len(s.RowsToCompensate()) == 0 {
		s.RaiseEvent(events.NewOrderAllocationRejected(orderId, commitId, s.reason))
	}
	return true
}

func (s *OrderAllocationSaga) rowInState(beerId string, state sagaRowState) bool {
	for _, row := range s.rows {
		if row.data.BeerId == beerId && row.state == state {
			return true
		}
	}
	return false
}

func (s *OrderAllocationSaga) allRowsIn(state sagaRowState) bool {
	for _, row := range s.rows {
		if row.state != state {
			return false
		}
	}
	return true
}

func (s *OrderAllocationSaga) setRowState(beerId string, state sagaRowState) {
	for i := range s.rows {
		if s.rows[i].data.BeerId == beerId {
			s.rows[i].state = state
		}
	}
}

// Route dispatches events to the apply methods — state changes only here,
// both on RaiseEvent and on stream replay.
func (s *OrderAllocationSaga) Route(event muflone.DomainEvent) {
	switch e := event.(type) {
	case events.OrderAllocationStarted:
		s.SetID(e.AggregateID())
		s.salesOrderId = e.SalesOrderId
		s.status = SagaRunning
		s.rows = make([]sagaRow, 0, len(e.Rows))
		for _, row := range e.Rows {
			s.rows = append(s.rows, sagaRow{data: row, state: rowPending})
		}
	case events.AllocationStepSucceeded:
		s.setRowState(e.BeerId, rowAllocated)
	case events.AllocationStepFailed:
		s.setRowState(e.BeerId, rowFailed)
		s.status = SagaCompensating
		s.reason = e.Reason
	case events.AllocationStepCompensated:
		s.setRowState(e.BeerId, rowCompensated)
	case events.OrderAllocationCompleted:
		s.status = SagaCompleted
	case events.OrderAllocationRejected:
		s.status = SagaRejected
		s.reason = e.Reason
	}
}
