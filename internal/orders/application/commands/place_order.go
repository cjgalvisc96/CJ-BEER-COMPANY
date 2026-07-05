// Package commands holds the write-side use cases of the orders context.
package commands

import (
	"context"
	"fmt"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type PlaceOrderInput struct {
	CustomerName string
	Lines        []PlaceOrderLine
}

type PlaceOrderLine struct {
	BeerID string
	Units  int
}

// PlaceOrderHandler validates every requested beer against the catalog
// (through the port), captures current prices into the order lines, and
// publishes OrderPlaced so inventory can attempt the stock reservation.
type PlaceOrderHandler struct {
	repository domain.OrderRepository
	catalog    ports.BeerCatalog
	publisher  sharedports.EventPublisher
}

func NewPlaceOrderHandler(
	repository domain.OrderRepository,
	catalog ports.BeerCatalog,
	publisher sharedports.EventPublisher,
) *PlaceOrderHandler {
	return &PlaceOrderHandler{repository: repository, catalog: catalog, publisher: publisher}
}

func (h *PlaceOrderHandler) Handle(ctx context.Context, input PlaceOrderInput) (dto.OrderOutput, error) {
	lines := make([]domain.OrderLine, 0, len(input.Lines))
	for _, requested := range input.Lines {
		beerID, err := domain.ParseBeerRef(requested.BeerID)
		if err != nil {
			return dto.OrderOutput{}, err
		}
		snapshot, err := h.catalog.FindBeer(ctx, requested.BeerID)
		if err != nil || !snapshot.Sellable {
			return dto.OrderOutput{}, fmt.Errorf("%w: %s", domain.ErrBeerNotSellable, requested.BeerID)
		}
		line, err := domain.NewOrderLine(beerID, requested.Units, snapshot.Price)
		if err != nil {
			return dto.OrderOutput{}, err
		}
		lines = append(lines, line)
	}

	order, err := domain.PlaceOrder(input.CustomerName, lines)
	if err != nil {
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
