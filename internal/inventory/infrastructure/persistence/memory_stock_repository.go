// Package persistence provides the in-memory adapter for the inventory
// repository port.
package persistence

import (
	"context"
	"sort"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
)

type stockRecord struct {
	beerID       domain.BeerRef
	quantity     int
	reorderLevel int
}

type MemoryStockRepository struct {
	mu    sync.RWMutex
	items map[string]stockRecord
}

func NewMemoryStockRepository() *MemoryStockRepository {
	return &MemoryStockRepository{items: make(map[string]stockRecord)}
}

func (r *MemoryStockRepository) Save(_ context.Context, item *domain.StockItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.BeerID().String()] = stockRecord{
		beerID:       item.BeerID(),
		quantity:     item.Quantity(),
		reorderLevel: item.ReorderLevel(),
	}
	return nil
}

func (r *MemoryStockRepository) FindByBeerID(_ context.Context, beerID domain.BeerRef) (*domain.StockItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.items[beerID.String()]
	if !ok {
		return nil, domain.ErrStockItemNotFound
	}
	return domain.RehydrateStockItem(record.beerID, record.quantity, record.reorderLevel), nil
}

func (r *MemoryStockRepository) FindAll(_ context.Context) ([]*domain.StockItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]*domain.StockItem, 0, len(r.items))
	for _, record := range r.items {
		items = append(items, domain.RehydrateStockItem(record.beerID, record.quantity, record.reorderLevel))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].BeerID().String() < items[j].BeerID().String()
	})
	return items, nil
}

func (r *MemoryStockRepository) Exists(_ context.Context, beerID domain.BeerRef) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.items[beerID.String()]
	return ok, nil
}
