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

// UpdateAvailabilityDueToSalesOrderCommandHandler allocates stock to a
// sales order.
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
		return err
	}
	if err := aggregate.UpdateDueToSalesOrder(command.CommitID(), command.Quantity); err != nil {
		// Allocation failures are business outcomes, not poison messages:
		// log and ack so the bus does not redeliver forever.
		if errors.Is(err, domain.ErrNotEnoughStock) {
			h.logger.Warn("warehouses.allocation_refused", slog.String("reason", err.Error()))
			return nil
		}
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}
