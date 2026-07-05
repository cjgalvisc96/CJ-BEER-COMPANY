// Package integration reacts to other contexts' integration events. The
// message struct is a consumer-driven contract: the Warehouses module
// deserializes only the fields it needs and never imports Sales types, so
// the two contexts stay decoupled (the book's Figure 4.2: SalesOrderCreated
// → the Warehouse context allocates stock).
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
)

// SalesOrderCreatedTopic is the integration topic published by the Sales
// module; the string is the contract, not a shared type.
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

// SalesOrderCreatedHandler turns each row of the order into an
// UpdateAvailabilityDueToSalesOrder command for the beer's aggregate.
type SalesOrderCreatedHandler struct {
	bus    *muflone.ServiceBus
	logger *slog.Logger
}

func NewSalesOrderCreatedHandler(bus *muflone.ServiceBus, logger *slog.Logger) *SalesOrderCreatedHandler {
	return &SalesOrderCreatedHandler{bus: bus, logger: logger}
}

func (h *SalesOrderCreatedHandler) Handle(ctx context.Context, payload []byte) error {
	var message salesOrderCreatedMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return fmt.Errorf("unmarshal %s: %w", SalesOrderCreatedTopic, err)
	}
	for _, row := range message.Rows {
		beerId, err := uuid.Parse(row.BeerId)
		if err != nil {
			h.logger.Warn("warehouses.integration.invalid_beer_id", slog.String("beer_id", row.BeerId))
			continue
		}
		command := commands.NewUpdateAvailabilityDueToSalesOrder(
			sharedkernel.BeerId{Value: beerId},
			message.CommitId,
			sharedkernel.BeerName{Value: row.BeerName},
			customtypes.NewQuantity(row.Quantity, row.UnitOfMeasure),
		)
		if err := h.bus.Send(ctx, command); err != nil {
			return err
		}
	}
	h.logger.Info("warehouses.integration.sales_order_processed",
		slog.String("sales_order_id", message.SalesOrderId))
	return nil
}
