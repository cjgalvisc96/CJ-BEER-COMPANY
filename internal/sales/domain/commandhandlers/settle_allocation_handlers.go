package commandhandlers

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/commands"
)

// MarkSalesOrderAllocatedCommandHandler settles the order after the
// warehouse saga completed.
type MarkSalesOrderAllocatedCommandHandler struct {
	repository muflone.Repository[*domain.SalesOrder]
	logger     *slog.Logger
}

func NewMarkSalesOrderAllocatedCommandHandler(
	repository muflone.Repository[*domain.SalesOrder],
	logger *slog.Logger,
) *MarkSalesOrderAllocatedCommandHandler {
	return &MarkSalesOrderAllocatedCommandHandler{repository: repository, logger: logger}
}

func (h *MarkSalesOrderAllocatedCommandHandler) Handle(ctx context.Context, command commands.MarkSalesOrderAllocated) error {
	aggregate, err := h.repository.GetByID(ctx, command.AggregateID())
	if err != nil {
		return err
	}
	changed, err := aggregate.MarkAllocated(command.CommitID())
	if err != nil || !changed {
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}

// MarkSalesOrderAllocationRejectedCommandHandler settles the order after
// the warehouse saga failed and compensated.
type MarkSalesOrderAllocationRejectedCommandHandler struct {
	repository muflone.Repository[*domain.SalesOrder]
	logger     *slog.Logger
}

func NewMarkSalesOrderAllocationRejectedCommandHandler(
	repository muflone.Repository[*domain.SalesOrder],
	logger *slog.Logger,
) *MarkSalesOrderAllocationRejectedCommandHandler {
	return &MarkSalesOrderAllocationRejectedCommandHandler{repository: repository, logger: logger}
}

func (h *MarkSalesOrderAllocationRejectedCommandHandler) Handle(
	ctx context.Context,
	command commands.MarkSalesOrderAllocationRejected,
) error {
	aggregate, err := h.repository.GetByID(ctx, command.AggregateID())
	if err != nil {
		return err
	}
	changed, err := aggregate.MarkAllocationRejected(command.CommitID(), command.Reason)
	if err != nil || !changed {
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}
