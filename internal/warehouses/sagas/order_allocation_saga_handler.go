// Package sagas hosts the order-allocation saga runner — the book's
// Chapter 12 realized: a CHOREOGRAPHED, EVENT-SOURCED saga. There is no
// central orchestrator across contexts (Sales and Warehouses only
// exchange integration events); within the warehouse, each allocation
// step is a local transaction on one Availability aggregate, driven by
// commands and advanced by the events those steps record. A failed step
// triggers idempotent COMPENSATING TRANSACTIONS (backward recovery) for
// every step already done, and the saga's own state is event-sourced in
// the `OrderAllocationSaga-<orderId>` stream, so it survives crashes
// (durable execution).
package sagas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/integrationevents"
)

// SalesOrderCreatedTopic is the integration topic published by the Sales
// module — the saga's trigger message. The string is the contract, not a
// shared type (consumer-driven).
const SalesOrderCreatedTopic = "sales.sales_order_created"

type salesOrderCreatedMessage struct {
	CommitId     uuid.UUID `json:"commit_id"`
	SalesOrderId string    `json:"sales_order_id"`
	Rows         []struct {
		BeerId        string `json:"beer_id"`
		BeerName      string `json:"beer_name"`
		Quantity      int    `json:"quantity"`
		UnitOfMeasure string `json:"unit_of_measure"`
	} `json:"rows"`
}

// OrderAllocationSaga coordinates the allocation of one sales order. The
// store powers durable execution: ListStreams finds in-flight sagas after
// a restart, ReadStream dates them for the step timeout.
type OrderAllocationSaga struct {
	repository muflone.Repository[*domain.OrderAllocationSaga]
	store      muflone.EventStore
	bus        *muflone.ServiceBus
	logger     *slog.Logger
}

func NewOrderAllocationSaga(
	repository muflone.Repository[*domain.OrderAllocationSaga],
	store muflone.EventStore,
	bus *muflone.ServiceBus,
	logger *slog.Logger,
) *OrderAllocationSaga {
	return &OrderAllocationSaga{repository: repository, store: store, bus: bus, logger: logger}
}

// OnSalesOrderCreated starts the saga. Redelivered triggers are idempotent:
// an already-started saga is left alone.
func (s *OrderAllocationSaga) OnSalesOrderCreated(ctx context.Context, payload []byte) error {
	var message salesOrderCreatedMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("unmarshal %s: %w", SalesOrderCreatedTopic, err)
	}
	salesOrderId, err := uuid.Parse(message.SalesOrderId)
	if err != nil {
		s.logger.Warn("warehouses.saga.invalid_sales_order_id", slog.String("id", message.SalesOrderId))
		return nil
	}
	if _, err := s.repository.GetByID(ctx, salesOrderId); err == nil {
		s.logger.Info("warehouses.saga.already_started", slog.String("sales_order_id", message.SalesOrderId))
		return nil
	} else if !errors.Is(err, muflone.ErrAggregateNotFound) {
		return err
	}

	rows := make([]events.SagaRowData, 0, len(message.Rows))
	for _, row := range message.Rows {
		rows = append(rows, events.SagaRowData(row))
	}
	saga := domain.StartOrderAllocation(salesOrderId, message.CommitId, rows)
	if err := s.repository.Save(ctx, saga, uuid.New()); err != nil {
		return err
	}
	if saga.Status() == domain.SagaCompleted { // an order with no rows
		return s.publishOutcome(ctx, saga, message.CommitId)
	}
	return s.sendNextAllocation(ctx, saga, message.CommitId)
}

// OnBeerAvailabilityUpdated advances the saga after a successful step.
func (s *OrderAllocationSaga) OnBeerAvailabilityUpdated(ctx context.Context, event events.BeerAvailabilityUpdated) error {
	saga, ok, err := s.load(ctx, event.SalesOrderId)
	if err != nil || !ok {
		return err
	}
	if !saga.RecordAllocationSucceeded(event.CommitID(), event.BeerId.Value.String()) {
		return nil // already observed (redelivery) — idempotent
	}
	if err := s.repository.Save(ctx, saga, uuid.New()); err != nil {
		return err
	}
	if saga.Status() == domain.SagaCompleted {
		return s.publishOutcome(ctx, saga, event.CommitID())
	}
	return s.sendNextAllocation(ctx, saga, event.CommitID())
}

// OnQuantityNotFound flips the saga into backward recovery: every
// already-allocated row gets a compensating transaction.
func (s *OrderAllocationSaga) OnQuantityNotFound(ctx context.Context, event events.QuantityNotFound) error {
	saga, ok, err := s.load(ctx, event.SalesOrderId)
	if err != nil || !ok {
		return err
	}
	reason := fmt.Sprintf("beer %s: requested %s, available %s",
		event.BeerId.Value, event.Requested.String(), event.Available.String())
	if !saga.RecordQuantityNotFound(event.CommitID(), event.BeerId.Value.String(), reason) {
		return nil
	}
	if err := s.repository.Save(ctx, saga, uuid.New()); err != nil {
		return err
	}
	if saga.Status() == domain.SagaRejected { // nothing was allocated yet
		return s.publishOutcome(ctx, saga, event.CommitID())
	}
	return s.sendCompensations(ctx, saga, event.CommitID())
}

// OnAvailabilityCompensated closes the loop of one compensating
// transaction; when the last one lands the saga rejects the order.
func (s *OrderAllocationSaga) OnAvailabilityCompensated(ctx context.Context, event events.AvailabilityCompensated) error {
	saga, ok, err := s.load(ctx, event.SalesOrderId)
	if err != nil || !ok {
		return err
	}
	if !saga.RecordCompensated(event.CommitID(), event.BeerId.Value.String()) {
		return nil
	}
	if err := s.repository.Save(ctx, saga, uuid.New()); err != nil {
		return err
	}
	if saga.Status() == domain.SagaRejected {
		return s.publishOutcome(ctx, saga, event.CommitID())
	}
	return nil
}

// load resolves a saga from a step event; events without an order (plain
// production flows) or for unknown sagas are ignored.
func (s *OrderAllocationSaga) load(ctx context.Context, salesOrderId string) (*domain.OrderAllocationSaga, bool, error) {
	if salesOrderId == "" {
		return nil, false, nil
	}
	orderId, err := uuid.Parse(salesOrderId)
	if err != nil {
		return nil, false, nil
	}
	saga, err := s.repository.GetByID(ctx, orderId)
	if err != nil {
		if errors.Is(err, muflone.ErrAggregateNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return saga, true, nil
}

func (s *OrderAllocationSaga) sendNextAllocation(ctx context.Context, saga *domain.OrderAllocationSaga, commitId uuid.UUID) error {
	// Only called while the saga runs, and a running saga always has a
	// pending row (see NextPendingRow); a broken invariant fails loudly on
	// the id parse below.
	row, _ := saga.NextPendingRow()
	beerId, err := uuid.Parse(row.BeerId)
	if err != nil {
		return fmt.Errorf("saga %s: invalid beer id %q", saga.SalesOrderId(), row.BeerId)
	}
	command := commands.NewUpdateAvailabilityDueToSalesOrder(
		sharedkernel.BeerId{Value: beerId},
		commitId,
		sharedkernel.BeerName{Value: row.BeerName},
		customtypes.NewQuantity(row.Quantity, row.UnitOfMeasure),
		saga.SalesOrderId(),
	)
	return s.bus.Send(ctx, command)
}

// sendCompensations dispatches the compensating transactions for every row
// still holding stock. Safe to re-drive: the Availability aggregate treats
// duplicate compensations as idempotent re-emissions.
func (s *OrderAllocationSaga) sendCompensations(ctx context.Context, saga *domain.OrderAllocationSaga, commitId uuid.UUID) error {
	for _, row := range saga.RowsToCompensate() {
		beerId, err := uuid.Parse(row.BeerId)
		if err != nil {
			return fmt.Errorf("saga %s: invalid beer id %q", saga.SalesOrderId(), row.BeerId)
		}
		command := commands.NewCompensateAvailabilityDueToFailedAllocation(
			sharedkernel.BeerId{Value: beerId},
			commitId,
			customtypes.NewQuantity(row.Quantity, row.UnitOfMeasure),
			saga.SalesOrderId(),
		)
		if err := s.bus.Send(ctx, command); err != nil {
			return err
		}
	}
	return nil
}

// ResumeInFlight re-drives every unfinished saga after a restart (book
// Ch. 12, durable execution): running sagas re-send their pending step,
// compensating sagas re-send their compensations. All downstream effects
// are idempotent, so resuming a saga whose outcome was already applied is
// harmless.
func (s *OrderAllocationSaga) ResumeInFlight(ctx context.Context) error {
	sagasInFlight, err := s.inFlight(ctx)
	if err != nil {
		return err
	}
	for _, saga := range sagasInFlight {
		s.logger.Info("warehouses.saga.resumed",
			slog.String("sales_order_id", saga.SalesOrderId()),
			slog.String("status", string(saga.Status())))
		if err := s.drive(ctx, saga, uuid.New()); err != nil {
			return err
		}
	}
	return nil
}

// TimeoutInFlight fails the pending step of every saga whose last recorded
// event is older than the cutoff (book Ch. 12: steps must not wait
// forever). Compensating sagas are re-driven instead — compensation is
// never abandoned.
func (s *OrderAllocationSaga) TimeoutInFlight(ctx context.Context, cutoff time.Time) error {
	sagasInFlight, err := s.inFlight(ctx)
	if err != nil {
		return err
	}
	for _, saga := range sagasInFlight {
		lastActivity, err := s.lastActivityOf(ctx, saga)
		if err != nil {
			return err
		}
		if !lastActivity.Before(cutoff) {
			continue
		}
		if saga.Status() == domain.SagaCompensating {
			s.logger.Warn("warehouses.saga.compensation_redriven",
				slog.String("sales_order_id", saga.SalesOrderId()))
			if err := s.sendCompensations(ctx, saga, uuid.New()); err != nil {
				return err
			}
			continue
		}
		row, _ := saga.NextPendingRow()
		s.logger.Warn("warehouses.saga.step_timed_out",
			slog.String("sales_order_id", saga.SalesOrderId()),
			slog.String("beer_id", row.BeerId))
		commitId := uuid.New()
		// A running saga always has a pending row, so the record always
		// transitions.
		saga.RecordQuantityNotFound(commitId, row.BeerId, "allocation step timed out")
		if err := s.repository.Save(ctx, saga, uuid.New()); err != nil {
			return err
		}
		if saga.Status() == domain.SagaRejected {
			if err := s.publishOutcome(ctx, saga, commitId); err != nil {
				return err
			}
			continue
		}
		if err := s.sendCompensations(ctx, saga, commitId); err != nil {
			return err
		}
	}
	return nil
}

// drive re-issues whatever an in-flight saga is waiting on.
func (s *OrderAllocationSaga) drive(ctx context.Context, saga *domain.OrderAllocationSaga, commitId uuid.UUID) error {
	if saga.Status() == domain.SagaCompensating {
		return s.sendCompensations(ctx, saga, commitId)
	}
	return s.sendNextAllocation(ctx, saga, commitId)
}

// inFlight loads every saga that is not settled yet.
func (s *OrderAllocationSaga) inFlight(ctx context.Context) ([]*domain.OrderAllocationSaga, error) {
	lister, ok := s.store.(muflone.StreamLister)
	if !ok {
		s.logger.Warn("warehouses.saga.store_cannot_list_streams")
		return nil, nil
	}
	streams, err := lister.ListStreams(ctx, domain.SagaStreamName)
	if err != nil {
		return nil, err
	}
	var inFlight []*domain.OrderAllocationSaga
	for _, streamID := range streams {
		orderId, err := uuid.Parse(strings.TrimPrefix(streamID, domain.SagaStreamName+"-"))
		if err != nil {
			s.logger.Warn("warehouses.saga.unparseable_stream", slog.String("stream", streamID))
			continue
		}
		saga, err := s.repository.GetByID(ctx, orderId)
		if err != nil {
			return nil, err
		}
		if saga.Status() == domain.SagaRunning || saga.Status() == domain.SagaCompensating {
			inFlight = append(inFlight, saga)
		}
	}
	return inFlight, nil
}

// lastActivityOf returns when the saga last recorded an event.
func (s *OrderAllocationSaga) lastActivityOf(ctx context.Context, saga *domain.OrderAllocationSaga) (time.Time, error) {
	orderId := uuid.MustParse(saga.SalesOrderId())
	stored, err := s.store.ReadStream(ctx, domain.SagaStreamName+"-"+orderId.String())
	if err != nil {
		return time.Time{}, err
	}
	return stored[len(stored)-1].OccurredAt, nil
}

func (s *OrderAllocationSaga) publishOutcome(ctx context.Context, saga *domain.OrderAllocationSaga, commitId uuid.UUID) error {
	if saga.Status() == domain.SagaCompleted {
		s.logger.Info("warehouses.saga.completed", slog.String("sales_order_id", saga.SalesOrderId()))
		return s.bus.PublishIntegrationEvent(ctx,
			integrationevents.NewOrderAllocationCompleted(saga.SalesOrderId(), commitId))
	}
	s.logger.Warn("warehouses.saga.rejected",
		slog.String("sales_order_id", saga.SalesOrderId()), slog.String("reason", saga.Reason()))
	return s.bus.PublishIntegrationEvent(ctx,
		integrationevents.NewOrderAllocationRejected(saga.SalesOrderId(), commitId, saga.Reason()))
}
