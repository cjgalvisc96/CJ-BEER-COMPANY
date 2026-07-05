// Package commandhandlers holds the Sales command handlers —
// BrewUp.Sales.Domain/CommandHandlers.
package commandhandlers

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/commands"
)

// CreateSalesOrderCommandHandler is the consumer for the CreateSalesOrder
// command: it invokes the aggregate factory and saves — the events raised
// in the factory are persisted by Repository.Save (the book's
// CreateSalesOrderCommandHandler.HandleAsync).
type CreateSalesOrderCommandHandler struct {
	repository muflone.Repository[*domain.SalesOrder]
	logger     *slog.Logger
}

func NewCreateSalesOrderCommandHandler(
	repository muflone.Repository[*domain.SalesOrder],
	logger *slog.Logger,
) *CreateSalesOrderCommandHandler {
	return &CreateSalesOrderCommandHandler{repository: repository, logger: logger}
}

func (h *CreateSalesOrderCommandHandler) Handle(ctx context.Context, command commands.CreateSalesOrder) error {
	aggregate, err := domain.CreateSalesOrder(
		command.SalesOrderId,
		command.CommitID(),
		command.SalesOrderNumber,
		command.OrderDate,
		command.CustomerId,
		command.CustomerName,
		command.Rows,
	)
	if err != nil {
		return err
	}
	return h.repository.Save(ctx, aggregate, uuid.New())
}
