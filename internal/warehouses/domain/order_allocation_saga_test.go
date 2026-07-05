package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

func twoRows() (events.SagaRowData, events.SagaRowData) {
	return events.SagaRowData{BeerId: uuid.NewString(), BeerName: "Plenty Lager", Quantity: 30, UnitOfMeasure: "Lt"},
		events.SagaRowData{BeerId: uuid.NewString(), BeerName: "Scarce Stout", Quantity: 50, UnitOfMeasure: "Lt"}
}

func TestSagaHappyPath(t *testing.T) {
	orderId, commitId := uuid.New(), uuid.New()
	rowA, rowB := twoRows()

	saga := domain.StartOrderAllocation(orderId, commitId, []events.SagaRowData{rowA, rowB})
	require.Equal(t, domain.SagaRunning, saga.Status())
	assert.Equal(t, orderId.String(), saga.SalesOrderId())

	next, ok := saga.NextPendingRow()
	require.True(t, ok)
	assert.Equal(t, rowA.BeerId, next.BeerId, "steps run in order, one at a time")

	require.True(t, saga.RecordAllocationSucceeded(commitId, rowA.BeerId))
	next, ok = saga.NextPendingRow()
	require.True(t, ok)
	assert.Equal(t, rowB.BeerId, next.BeerId)

	require.True(t, saga.RecordAllocationSucceeded(commitId, rowB.BeerId))
	assert.Equal(t, domain.SagaCompleted, saga.Status())
	_, ok = saga.NextPendingRow()
	assert.False(t, ok, "a finished saga has no pending steps")
}

func TestSagaCompensatesOnFailure(t *testing.T) {
	orderId, commitId := uuid.New(), uuid.New()
	rowA, rowB := twoRows()
	saga := domain.StartOrderAllocation(orderId, commitId, []events.SagaRowData{rowA, rowB})
	require.True(t, saga.RecordAllocationSucceeded(commitId, rowA.BeerId))

	require.True(t, saga.RecordQuantityNotFound(commitId, rowB.BeerId, "not enough stout"))

	assert.Equal(t, domain.SagaCompensating, saga.Status())
	toCompensate := saga.RowsToCompensate()
	require.Len(t, toCompensate, 1, "only the allocated row is compensated")
	assert.Equal(t, rowA.BeerId, toCompensate[0].BeerId)

	require.True(t, saga.RecordCompensated(commitId, rowA.BeerId))
	assert.Equal(t, domain.SagaRejected, saga.Status())
	assert.Equal(t, "not enough stout", saga.Reason())
}

func TestSagaRejectsImmediatelyWithNothingToCompensate(t *testing.T) {
	orderId, commitId := uuid.New(), uuid.New()
	rowA, _ := twoRows()
	saga := domain.StartOrderAllocation(orderId, commitId, []events.SagaRowData{rowA})

	require.True(t, saga.RecordQuantityNotFound(commitId, rowA.BeerId, "empty warehouse"))

	assert.Equal(t, domain.SagaRejected, saga.Status())
	assert.Empty(t, saga.RowsToCompensate())
}

func TestSagaWithNoRowsCompletesImmediately(t *testing.T) {
	saga := domain.StartOrderAllocation(uuid.New(), uuid.New(), nil)

	assert.Equal(t, domain.SagaCompleted, saga.Status())
}

// TestSagaRecordMethodsAreIdempotent: redelivered messages re-observe
// known facts and change nothing (book Ch. 12: durable execution requires
// idempotent steps).
func TestSagaRecordMethodsAreIdempotent(t *testing.T) {
	orderId, commitId := uuid.New(), uuid.New()
	rowA, rowB := twoRows()
	saga := domain.StartOrderAllocation(orderId, commitId, []events.SagaRowData{rowA, rowB})

	require.True(t, saga.RecordAllocationSucceeded(commitId, rowA.BeerId))
	assert.False(t, saga.RecordAllocationSucceeded(commitId, rowA.BeerId), "duplicate step event")
	assert.False(t, saga.RecordCompensated(commitId, rowA.BeerId), "not compensating yet")

	require.True(t, saga.RecordQuantityNotFound(commitId, rowB.BeerId, "shortage"))
	assert.False(t, saga.RecordQuantityNotFound(commitId, rowB.BeerId, "shortage"), "duplicate failure")
	assert.False(t, saga.RecordAllocationSucceeded(commitId, rowB.BeerId), "saga is compensating")

	require.True(t, saga.RecordCompensated(commitId, rowA.BeerId))
	assert.False(t, saga.RecordCompensated(commitId, rowA.BeerId), "duplicate compensation")
	assert.Equal(t, domain.SagaRejected, saga.Status())
	assert.False(t, saga.RecordQuantityNotFound(commitId, rowA.BeerId, "late"), "saga is settled")
}

// TestSagaReplaysFromItsStream: the saga is event-sourced — replaying the
// recorded events rebuilds the exact state (durable execution).
func TestSagaReplaysFromItsStream(t *testing.T) {
	orderId, commitId := uuid.New(), uuid.New()
	rowA, rowB := twoRows()
	saga := domain.StartOrderAllocation(orderId, commitId, []events.SagaRowData{rowA, rowB})
	require.True(t, saga.RecordAllocationSucceeded(commitId, rowA.BeerId))
	require.True(t, saga.RecordQuantityNotFound(commitId, rowB.BeerId, "shortage"))
	stream := saga.UncommittedEvents()

	replayed := domain.NewOrderAllocationSaga()
	for _, event := range stream {
		replayed.ApplyEvent(event)
	}

	assert.Equal(t, domain.SagaCompensating, replayed.Status())
	assert.Equal(t, saga.RowsToCompensate(), replayed.RowsToCompensate())
	assert.Equal(t, saga.Reason(), replayed.Reason())
}
