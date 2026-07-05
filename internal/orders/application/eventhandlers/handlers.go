// Package eventhandlers closes the order-fulfilment loop: it reacts to the
// inventory context's reservation outcome. Message structs are
// consumer-driven contracts (see the inventory eventhandlers package).
package eventhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
)

const (
	orderStockReservedTopic = "inventory.order_stock_reserved"
	orderStockRejectedTopic = "inventory.order_stock_rejected"
)

type stockReservedMessage struct {
	OrderID string `json:"order_id"`
}

type stockRejectedMessage struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

type Handlers struct {
	settle *commands.SettleOrderHandler
	logger *slog.Logger
}

func NewHandlers(settle *commands.SettleOrderHandler, logger *slog.Logger) *Handlers {
	return &Handlers{settle: settle, logger: logger}
}

func (h *Handlers) StockReservedTopic() string { return orderStockReservedTopic }
func (h *Handlers) StockRejectedTopic() string { return orderStockRejectedTopic }

func (h *Handlers) OnStockReserved(ctx context.Context, payload []byte) error {
	var msg stockReservedMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("unmarshal %s: %w", orderStockReservedTopic, err)
	}
	if err := h.settle.Confirm(ctx, msg.OrderID); err != nil {
		// The order may have been cancelled while the reservation was in
		// flight; that race is expected, not a processing failure.
		if errors.Is(err, domain.ErrOrderNotPending) {
			h.logger.Warn("orders.confirm_skipped", slog.String("order_id", msg.OrderID))
			return nil
		}
		return fmt.Errorf("confirm order %s: %w", msg.OrderID, err)
	}
	h.logger.Info("orders.confirmed", slog.String("order_id", msg.OrderID))
	return nil
}

func (h *Handlers) OnStockRejected(ctx context.Context, payload []byte) error {
	var msg stockRejectedMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("unmarshal %s: %w", orderStockRejectedTopic, err)
	}
	if err := h.settle.Reject(ctx, msg.OrderID, msg.Reason); err != nil {
		if errors.Is(err, domain.ErrOrderNotPending) {
			h.logger.Warn("orders.reject_skipped", slog.String("order_id", msg.OrderID))
			return nil
		}
		return fmt.Errorf("reject order %s: %w", msg.OrderID, err)
	}
	h.logger.Info("orders.rejected",
		slog.String("order_id", msg.OrderID), slog.String("reason", msg.Reason))
	return nil
}
