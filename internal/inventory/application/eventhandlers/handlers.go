// Package eventhandlers reacts to events published by other contexts.
//
// The message structs here are consumer-driven contracts: local copies of
// just the fields inventory needs, deserialized from the JSON on the wire.
// Inventory never imports another context's domain types, so contexts stay
// deployable and evolvable independently.
package eventhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/commands"
)

// Topics inventory subscribes to. The strings are the public contract of
// the producing contexts.
const (
	batchCompletedTopic = "brewing.batch_completed"
	orderPlacedTopic    = "orders.order_placed"
)

type batchCompletedMessage struct {
	BeerID string `json:"beer_id"`
	Units  int    `json:"units"`
}

type orderPlacedMessage struct {
	OrderID string `json:"order_id"`
	Lines   []struct {
		BeerID string `json:"beer_id"`
		Units  int    `json:"units"`
	} `json:"lines"`
}

// Handlers groups the inventory reactions so the module can register them
// on the bus in one place.
type Handlers struct {
	replenish *commands.ReplenishStockHandler
	reserve   *commands.ReserveOrderStockHandler
	logger    *slog.Logger
}

func NewHandlers(
	replenish *commands.ReplenishStockHandler,
	reserve *commands.ReserveOrderStockHandler,
	logger *slog.Logger,
) *Handlers {
	return &Handlers{replenish: replenish, reserve: reserve, logger: logger}
}

func (h *Handlers) BatchCompletedTopic() string { return batchCompletedTopic }
func (h *Handlers) OrderPlacedTopic() string    { return orderPlacedTopic }

// OnBatchCompleted moves finished production into sellable stock.
func (h *Handlers) OnBatchCompleted(ctx context.Context, payload []byte) error {
	var msg batchCompletedMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("unmarshal %s: %w", batchCompletedTopic, err)
	}
	_, err := h.replenish.Handle(ctx, commands.ReplenishStockInput{
		BeerID: msg.BeerID,
		Units:  msg.Units,
	})
	if err != nil {
		return fmt.Errorf("replenish stock for beer %s: %w", msg.BeerID, err)
	}
	h.logger.Info("inventory.replenished_from_batch",
		slog.String("beer_id", msg.BeerID), slog.Int("units", msg.Units))
	return nil
}

// OnOrderPlaced tries to reserve stock for the order and publishes the
// outcome; the orders context reacts to that outcome, completing the
// choreography.
func (h *Handlers) OnOrderPlaced(ctx context.Context, payload []byte) error {
	var msg orderPlacedMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("unmarshal %s: %w", orderPlacedTopic, err)
	}
	input := commands.ReserveOrderStockInput{OrderID: msg.OrderID}
	for _, line := range msg.Lines {
		input.Lines = append(input.Lines, commands.ReserveOrderStockLine{
			BeerID: line.BeerID,
			Units:  line.Units,
		})
	}
	if err := h.reserve.Handle(ctx, input); err != nil {
		return fmt.Errorf("reserve stock for order %s: %w", msg.OrderID, err)
	}
	h.logger.Info("inventory.order_reservation_processed", slog.String("order_id", msg.OrderID))
	return nil
}
