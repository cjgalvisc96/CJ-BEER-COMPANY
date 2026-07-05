// Package integration reacts to other contexts' integration events with
// consumer-driven contracts: the allocation-saga outcome from the
// warehouse becomes a Sales command settling the order (the book's
// Ch. 12, Figure 12.3: the order service updates the order status).
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/commands"
)

const (
	OrderAllocationCompletedTopic = "warehouses.order_allocation_completed"
	OrderAllocationRejectedTopic  = "warehouses.order_allocation_rejected"
)

type allocationOutcomeMessage struct {
	CommitId     uuid.UUID `json:"commit_id"`
	SalesOrderId string    `json:"sales_order_id"`
	Reason       string    `json:"reason"`
}

// AllocationOutcomeHandler turns the saga outcome into Sales commands.
type AllocationOutcomeHandler struct {
	bus    *muflone.ServiceBus
	logger *slog.Logger
}

func NewAllocationOutcomeHandler(bus *muflone.ServiceBus, logger *slog.Logger) *AllocationOutcomeHandler {
	return &AllocationOutcomeHandler{bus: bus, logger: logger}
}

func (h *AllocationOutcomeHandler) OnAllocationCompleted(ctx context.Context, payload []byte) error {
	message, err := h.parse(OrderAllocationCompletedTopic, payload)
	if err != nil {
		return err
	}
	if message == nil {
		return nil
	}
	h.logger.Info("sales.allocation_completed", slog.String("sales_order_id", message.SalesOrderId))
	return h.bus.Send(ctx, commands.NewMarkSalesOrderAllocated(
		sharedkernel.SalesOrderId{Value: uuid.MustParse(message.SalesOrderId)}, message.CommitId,
	))
}

func (h *AllocationOutcomeHandler) OnAllocationRejected(ctx context.Context, payload []byte) error {
	message, err := h.parse(OrderAllocationRejectedTopic, payload)
	if err != nil {
		return err
	}
	if message == nil {
		return nil
	}
	h.logger.Warn("sales.allocation_rejected",
		slog.String("sales_order_id", message.SalesOrderId), slog.String("reason", message.Reason))
	return h.bus.Send(ctx, commands.NewMarkSalesOrderAllocationRejected(
		sharedkernel.SalesOrderId{Value: uuid.MustParse(message.SalesOrderId)}, message.CommitId, message.Reason,
	))
}

// parse returns (nil, nil) for messages Sales cannot act on — they are
// logged and acked, not poison.
func (h *AllocationOutcomeHandler) parse(topic string, payload []byte) (*allocationOutcomeMessage, error) {
	var message allocationOutcomeMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", topic, err)
	}
	if _, err := uuid.Parse(message.SalesOrderId); err != nil {
		h.logger.Warn("sales.integration.invalid_sales_order_id", slog.String("id", message.SalesOrderId))
		return nil, nil
	}
	return &message, nil
}
