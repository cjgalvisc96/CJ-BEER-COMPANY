// Package queries holds the read-side use cases of the orders context.
package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
)

type GetOrderHandler struct {
	repository domain.OrderRepository
}

func NewGetOrderHandler(repository domain.OrderRepository) *GetOrderHandler {
	return &GetOrderHandler{repository: repository}
}

func (h *GetOrderHandler) Handle(ctx context.Context, rawOrderID string) (dto.OrderOutput, error) {
	id, err := domain.ParseOrderID(rawOrderID)
	if err != nil {
		return dto.OrderOutput{}, err
	}
	order, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.OrderOutput{}, err
	}
	return dto.OrderOutputFromEntity(order)
}
