package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type CancelOrderHandler struct {
	repository domain.OrderRepository
	publisher  sharedports.EventPublisher
}

func NewCancelOrderHandler(repository domain.OrderRepository, publisher sharedports.EventPublisher) *CancelOrderHandler {
	return &CancelOrderHandler{repository: repository, publisher: publisher}
}

func (h *CancelOrderHandler) Handle(ctx context.Context, rawOrderID string) (dto.OrderOutput, error) {
	id, err := domain.ParseOrderID(rawOrderID)
	if err != nil {
		return dto.OrderOutput{}, err
	}
	order, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.OrderOutput{}, err
	}
	if err := order.Cancel(); err != nil {
		return dto.OrderOutput{}, err
	}
	if err := h.repository.Save(ctx, order); err != nil {
		return dto.OrderOutput{}, err
	}
	if err := h.publisher.Publish(ctx, order.PullEvents()...); err != nil {
		return dto.OrderOutput{}, err
	}
	return dto.OrderOutputFromEntity(order)
}
