package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
)

type ListStockHandler struct {
	repository domain.StockRepository
}

func NewListStockHandler(repository domain.StockRepository) *ListStockHandler {
	return &ListStockHandler{repository: repository}
}

func (h *ListStockHandler) Handle(ctx context.Context) ([]dto.StockOutput, error) {
	items, err := h.repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	outputs := make([]dto.StockOutput, 0, len(items))
	for _, item := range items {
		outputs = append(outputs, dto.StockOutputFromEntity(item))
	}
	return outputs, nil
}
