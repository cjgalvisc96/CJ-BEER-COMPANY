// Package eventhandlers holds the Sales read-model projections —
// BrewUp.Sales.ReadModel/EventHandlers. Each handler reacts to a domain
// event and updates a projection through the read-model service.
package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

// salesOrderWriter is the slice of the read-model service this projection
// needs (the book's ISalesOrderService).
type salesOrderWriter interface {
	CreateSalesOrder(ctx context.Context, order dtos.SalesOrder) error
}

// SalesOrderCreatedEventHandler projects the new order into the read model
// (the book's SalesOrderCreatedEventHandlerAsync).
type SalesOrderCreatedEventHandler struct {
	service salesOrderWriter
	logger  *slog.Logger
}

func NewSalesOrderCreatedEventHandler(service salesOrderWriter, logger *slog.Logger) *SalesOrderCreatedEventHandler {
	return &SalesOrderCreatedEventHandler{service: service, logger: logger}
}

func (h *SalesOrderCreatedEventHandler) Handle(ctx context.Context, event events.SalesOrderCreated) error {
	rows := make([]dtos.SalesOrderRow, 0, len(event.Rows))
	for _, row := range event.Rows {
		rows = append(rows, dtos.SalesOrderRow{
			BeerId:   row.BeerId.Value.String(),
			BeerName: row.BeerName.Value,
			Quantity: row.Quantity,
			Price:    row.Price,
		})
	}
	if err := h.service.CreateSalesOrder(ctx, dtos.SalesOrder{
		Id:               event.SalesOrderId.Value.String(),
		SalesOrderNumber: event.SalesOrderNumber.Value,
		OrderDate:        event.OrderDate.Value,
		CustomerId:       event.CustomerId.Value.String(),
		CustomerName:     event.CustomerName.Value,
		Rows:             rows,
	}); err != nil {
		return err
	}
	h.logger.Info("sales.readmodel.sales_order_projected",
		slog.String("sales_order_id", event.SalesOrderId.Value.String()))
	return nil
}
