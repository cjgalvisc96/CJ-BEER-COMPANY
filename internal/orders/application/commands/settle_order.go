package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

// SettleOrderHandler finishes the reservation process: it confirms or
// rejects a pending order depending on the inventory outcome. It is driven
// by events, not by HTTP.
type SettleOrderHandler struct {
	repository domain.OrderRepository
	publisher  sharedports.EventPublisher
}

func NewSettleOrderHandler(repository domain.OrderRepository, publisher sharedports.EventPublisher) *SettleOrderHandler {
	return &SettleOrderHandler{repository: repository, publisher: publisher}
}

func (h *SettleOrderHandler) Confirm(ctx context.Context, rawOrderID string) error {
	return h.settle(ctx, rawOrderID, func(order *domain.Order) error {
		return order.Confirm()
	})
}

func (h *SettleOrderHandler) Reject(ctx context.Context, rawOrderID, reason string) error {
	return h.settle(ctx, rawOrderID, func(order *domain.Order) error {
		return order.Reject(reason)
	})
}

func (h *SettleOrderHandler) settle(
	ctx context.Context,
	rawOrderID string,
	transition func(*domain.Order) error,
) error {
	id, err := domain.ParseOrderID(rawOrderID)
	if err != nil {
		return err
	}
	order, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := transition(order); err != nil {
		return err
	}
	if err := h.repository.Save(ctx, order); err != nil {
		return err
	}
	return h.publisher.Publish(ctx, order.PullEvents()...)
}
