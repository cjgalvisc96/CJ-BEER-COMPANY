// Package commands holds the write-side use cases of the inventory context.
package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
)

type TrackStockItemInput struct {
	BeerID       string
	ReorderLevel int
}

// TrackStockItemHandler starts tracking inventory for a beer at zero stock.
type TrackStockItemHandler struct {
	repository domain.StockRepository
}

func NewTrackStockItemHandler(repository domain.StockRepository) *TrackStockItemHandler {
	return &TrackStockItemHandler{repository: repository}
}

func (h *TrackStockItemHandler) Handle(ctx context.Context, input TrackStockItemInput) (dto.StockOutput, error) {
	beerID, err := domain.ParseBeerRef(input.BeerID)
	if err != nil {
		return dto.StockOutput{}, err
	}
	exists, err := h.repository.Exists(ctx, beerID)
	if err != nil {
		return dto.StockOutput{}, err
	}
	if exists {
		return dto.StockOutput{}, domain.ErrStockItemAlreadyExists
	}
	item, err := domain.NewStockItem(beerID, input.ReorderLevel)
	if err != nil {
		return dto.StockOutput{}, err
	}
	if err := h.repository.Save(ctx, item); err != nil {
		return dto.StockOutput{}, err
	}
	return dto.StockOutputFromEntity(item), nil
}
