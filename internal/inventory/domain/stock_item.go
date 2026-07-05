// Package domain models the inventory bounded context: how much of each
// beer is available in the warehouse.
package domain

import (
	"fmt"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// BeerRef is an opaque reference to a beer in the catalog context. Holding
// only the identifier (not the Beer entity) keeps the contexts isolated.
type BeerRef struct {
	shared.EntityID
}

func ParseBeerRef(raw string) (BeerRef, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return BeerRef{}, err
	}
	return BeerRef{EntityID: id}, nil
}

// StockItem is the aggregate root: the stock level of one beer. The
// invariant is simple and absolute — quantity never goes below zero.
type StockItem struct {
	shared.AggregateRoot

	beerID       BeerRef
	quantity     int
	reorderLevel int
}

// NewStockItem starts tracking a beer at zero stock.
func NewStockItem(beerID BeerRef, reorderLevel int) (*StockItem, error) {
	if reorderLevel < 0 {
		return nil, shared.NewValidationError("reorder level cannot be negative")
	}
	return &StockItem{beerID: beerID, reorderLevel: reorderLevel}, nil
}

// RehydrateStockItem rebuilds a StockItem from persisted state.
func RehydrateStockItem(beerID BeerRef, quantity, reorderLevel int) *StockItem {
	return &StockItem{beerID: beerID, quantity: quantity, reorderLevel: reorderLevel}
}

func (s *StockItem) BeerID() BeerRef   { return s.beerID }
func (s *StockItem) Quantity() int     { return s.quantity }
func (s *StockItem) ReorderLevel() int { return s.reorderLevel }

// Replenish adds produced units and records StockReplenished.
func (s *StockItem) Replenish(units int) error {
	if units <= 0 {
		return shared.NewValidationError("replenish units must be positive")
	}
	s.quantity += units
	s.RecordEvent(StockReplenished{
		BaseEvent: shared.NewBaseEvent(),
		BeerID:    s.beerID.String(),
		Units:     units,
		Quantity:  s.quantity,
	})
	return nil
}

// Reserve removes units for an order. It fails without side effects when
// there is not enough stock, and warns (StockLevelLow) when the remaining
// quantity crosses the reorder level.
func (s *StockItem) Reserve(units int) error {
	if units <= 0 {
		return shared.NewValidationError("reserve units must be positive")
	}
	if units > s.quantity {
		return fmt.Errorf(
			"%w: beer %s has %d units, requested %d",
			ErrInsufficientStock, s.beerID.String(), s.quantity, units,
		)
	}
	s.quantity -= units
	s.RecordEvent(StockReserved{
		BaseEvent: shared.NewBaseEvent(),
		BeerID:    s.beerID.String(),
		Units:     units,
		Quantity:  s.quantity,
	})
	if s.quantity <= s.reorderLevel {
		s.RecordEvent(StockLevelLow{
			BaseEvent:    shared.NewBaseEvent(),
			BeerID:       s.beerID.String(),
			Quantity:     s.quantity,
			ReorderLevel: s.reorderLevel,
		})
	}
	return nil
}
