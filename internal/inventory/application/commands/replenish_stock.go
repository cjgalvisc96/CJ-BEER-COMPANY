package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type ReplenishStockInput struct {
	BeerID string
	Units  int
}

// ReplenishStockHandler adds units to a beer's stock. If the beer is not
// tracked yet it starts tracking it implicitly — production output must
// never be dropped on the floor.
type ReplenishStockHandler struct {
	repository domain.StockRepository
	publisher  ports.EventPublisher
}

func NewReplenishStockHandler(repository domain.StockRepository, publisher ports.EventPublisher) *ReplenishStockHandler {
	return &ReplenishStockHandler{repository: repository, publisher: publisher}
}

func (h *ReplenishStockHandler) Handle(ctx context.Context, input ReplenishStockInput) (dto.StockOutput, error) {
	beerID, err := domain.ParseBeerRef(input.BeerID)
	if err != nil {
		return dto.StockOutput{}, err
	}
	item, err := h.repository.FindByBeerID(ctx, beerID)
	if err != nil {
		item, err = domain.NewStockItem(beerID, 0)
		if err != nil {
			return dto.StockOutput{}, err
		}
	}
	if err := item.Replenish(input.Units); err != nil {
		return dto.StockOutput{}, err
	}
	if err := h.repository.Save(ctx, item); err != nil {
		return dto.StockOutput{}, err
	}
	if err := h.publisher.Publish(ctx, item.PullEvents()...); err != nil {
		return dto.StockOutput{}, err
	}
	return dto.StockOutputFromEntity(item), nil
}
