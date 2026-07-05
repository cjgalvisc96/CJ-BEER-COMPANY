// Package queries holds the read-side use cases of the inventory context.
package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
)

type GetStockHandler struct {
	repository domain.StockRepository
}

func NewGetStockHandler(repository domain.StockRepository) *GetStockHandler {
	return &GetStockHandler{repository: repository}
}

func (h *GetStockHandler) Handle(ctx context.Context, rawBeerID string) (dto.StockOutput, error) {
	beerID, err := domain.ParseBeerRef(rawBeerID)
	if err != nil {
		return dto.StockOutput{}, err
	}
	item, err := h.repository.FindByBeerID(ctx, beerID)
	if err != nil {
		return dto.StockOutput{}, err
	}
	return dto.StockOutputFromEntity(item), nil
}
