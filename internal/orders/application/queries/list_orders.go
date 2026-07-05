package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
)

type ListOrdersHandler struct {
	repository domain.OrderRepository
}

func NewListOrdersHandler(repository domain.OrderRepository) *ListOrdersHandler {
	return &ListOrdersHandler{repository: repository}
}

func (h *ListOrdersHandler) Handle(ctx context.Context) ([]dto.OrderOutput, error) {
	orders, err := h.repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	outputs := make([]dto.OrderOutput, 0, len(orders))
	for _, order := range orders {
		output, err := dto.OrderOutputFromEntity(order)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}
