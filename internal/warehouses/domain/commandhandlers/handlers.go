// Package commandhandlers holds the Warehouses command handlers —
// BrewUp.Warehouses.Domain/CommandHandlers.
package commandhandlers

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
)

// UpdateAvailabilityDueToProductionOrderCommandHandler is the book's
// handler of the same name: it loads the beer's Availability (or starts
// tracking it on the first production order) and adds the produced
// quantity.
type UpdateAvailabilityDueToProductionOrderCommandHandler struct {
	repository muflone.Repository[*domain.Availability]
	logger     *slog.Logger
}

func NewUpdateAvailabilityDueToProductionOrderCommandHandler(
	repository muflone.Repository[*domain.Availability],
	logger *slog.Logger,
) *UpdateAvailabilityDueToProductionOrderCommandHandler {
	return &UpdateAvailabilityDueToProductionOrderCommandHandler{repository: repository, logger: logger}
}

func (h *UpdateAvailabilityDueToProductionOrderCommandHandler) Handle(
	ctx context.Context,
	command commands.UpdateAvailabilityDueToProductionOrder,
) error {
	aggregate, err := h.repository.GetByID(ctx, command.AggregateID())
	if err != nil {
		if !errors.Is(err, muflone.ErrAggregateNotFound) {
			return err
		}
		aggregate, err = domain.CreateAvailability(
			command.BeerId, command.CommitID(), command.BeerName, command.Quantity,
		)
		if err != nil {
			return err
		}
		return h.repository.Save(ctx, aggregate, uuid.New())
	}
	if err := aggregate.UpdateDueToProductionOrder(command.CommitID(), command.BeerName, command.Quantity); err != nil {
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}

// UpdateAvailabilityDueToSalesOrderCommandHandler executes one saga step.
// The outcome — BeerAvailabilityUpdated or QuantityNotFound (the book's
// Ch. 12 failure event) — is recorded on the aggregate and published; the
// saga reacts to it.
type UpdateAvailabilityDueToSalesOrderCommandHandler struct {
	repository muflone.Repository[*domain.Availability]
	logger     *slog.Logger
}

func NewUpdateAvailabilityDueToSalesOrderCommandHandler(
	repository muflone.Repository[*domain.Availability],
	logger *slog.Logger,
) *UpdateAvailabilityDueToSalesOrderCommandHandler {
	return &UpdateAvailabilityDueToSalesOrderCommandHandler{repository: repository, logger: logger}
}

func (h *UpdateAvailabilityDueToSalesOrderCommandHandler) Handle(
	ctx context.Context,
	command commands.UpdateAvailabilityDueToSalesOrder,
) error {
	aggregate, err := h.repository.GetByID(ctx, command.AggregateID())
	if err != nil {
		if !errors.Is(err, muflone.ErrAggregateNotFound) {
			return err
		}
		// An untracked beer has zero availability: record the refusal so
		// the saga fails the step instead of hanging.
		refusal := domain.RefuseUnknownBeer(
			command.BeerId, command.CommitID(), command.Quantity, command.SalesOrderId,
		)
		return h.repository.Save(ctx, refusal, uuid.New())
	}
	if err := aggregate.UpdateDueToSalesOrder(command.CommitID(), command.Quantity, command.SalesOrderId); err != nil {
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}

// CompensateAvailabilityDueToFailedAllocationCommandHandler executes the
// saga's compensating transaction.
type CompensateAvailabilityDueToFailedAllocationCommandHandler struct {
	repository muflone.Repository[*domain.Availability]
	logger     *slog.Logger
}

func NewCompensateAvailabilityDueToFailedAllocationCommandHandler(
	repository muflone.Repository[*domain.Availability],
	logger *slog.Logger,
) *CompensateAvailabilityDueToFailedAllocationCommandHandler {
	return &CompensateAvailabilityDueToFailedAllocationCommandHandler{repository: repository, logger: logger}
}

func (h *CompensateAvailabilityDueToFailedAllocationCommandHandler) Handle(
	ctx context.Context,
	command commands.CompensateAvailabilityDueToFailedAllocation,
) error {
	aggregate, err := h.repository.GetByID(ctx, command.AggregateID())
	if err != nil {
		return err
	}
	if err := aggregate.CompensateDueToFailedAllocation(command.CommitID(), command.Quantity, command.SalesOrderId); err != nil {
		return err
	}
	h.logger.Info("warehouses.compensation_applied",
		slog.String("beer_id", command.BeerId.Value.String()),
		slog.String("sales_order_id", command.SalesOrderId))
	return h.repository.Save(ctx, aggregate, uuid.New())
}
