package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/integration"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type ReserveOrderStockInput struct {
	OrderID string
	Lines   []ReserveOrderStockLine
}

type ReserveOrderStockLine struct {
	BeerID string
	Units  int
}

// ReserveOrderStockHandler reserves stock for every line of an order
// atomically-in-effect: it verifies all lines first, then applies the
// reservations, and always publishes the outcome (reserved or rejected) so
// the orders context can finish its process.
type ReserveOrderStockHandler struct {
	repository domain.StockRepository
	publisher  ports.EventPublisher
}

func NewReserveOrderStockHandler(repository domain.StockRepository, publisher ports.EventPublisher) *ReserveOrderStockHandler {
	return &ReserveOrderStockHandler{repository: repository, publisher: publisher}
}

func (h *ReserveOrderStockHandler) Handle(ctx context.Context, input ReserveOrderStockInput) error {
	items, err := h.loadAndVerify(ctx, input.Lines)
	if err != nil {
		if _, isDomain := shared.KindOf(err); isDomain {
			return h.publisher.Publish(ctx, integration.OrderStockRejected{
				BaseEvent: shared.NewBaseEvent(),
				OrderID:   input.OrderID,
				Reason:    err.Error(),
			})
		}
		return err
	}

	events := make([]shared.Event, 0, len(items))
	for i, item := range items {
		if err := item.Reserve(input.Lines[i].Units); err != nil {
			return err
		}
		if err := h.repository.Save(ctx, item); err != nil {
			return err
		}
		events = append(events, item.PullEvents()...)
	}
	events = append(events, integration.OrderStockReserved{
		BaseEvent: shared.NewBaseEvent(),
		OrderID:   input.OrderID,
	})
	return h.publisher.Publish(ctx, events...)
}

// loadAndVerify returns the stock items for all lines, or a domain error if
// any line cannot be fulfilled. Nothing is mutated here.
func (h *ReserveOrderStockHandler) loadAndVerify(
	ctx context.Context,
	lines []ReserveOrderStockLine,
) ([]*domain.StockItem, error) {
	items := make([]*domain.StockItem, 0, len(lines))
	for _, line := range lines {
		beerID, err := domain.ParseBeerRef(line.BeerID)
		if err != nil {
			return nil, err
		}
		item, err := h.repository.FindByBeerID(ctx, beerID)
		if err != nil {
			if errors.Is(err, domain.ErrStockItemNotFound) {
				return nil, fmt.Errorf("%w: beer %s is not stocked", domain.ErrInsufficientStock, line.BeerID)
			}
			return nil, err
		}
		if line.Units <= 0 {
			return nil, shared.NewValidationError("reserve units must be positive")
		}
		if item.Quantity() < line.Units {
			return nil, fmt.Errorf(
				"%w: beer %s has %d units, requested %d",
				domain.ErrInsufficientStock, line.BeerID, item.Quantity(), line.Units,
			)
		}
		items = append(items, item)
	}
	return items, nil
}
