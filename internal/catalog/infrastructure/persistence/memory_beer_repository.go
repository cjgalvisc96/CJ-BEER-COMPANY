// Package persistence provides the in-memory adapter for the catalog
// repository port. Swapping to Postgres means adding another file here; the
// domain and application layers stay untouched.
package persistence

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// beerRecord is the persistence model: a snapshot of the aggregate state,
// so stored data cannot be mutated through shared pointers.
type beerRecord struct {
	id          domain.BeerID
	name        string
	style       domain.Style
	abv         domain.ABV
	price       shared.Money
	description string
	status      domain.BeerStatus
}

type MemoryBeerRepository struct {
	mu    sync.RWMutex
	beers map[string]beerRecord
}

func NewMemoryBeerRepository() *MemoryBeerRepository {
	return &MemoryBeerRepository{beers: make(map[string]beerRecord)}
}

func (r *MemoryBeerRepository) Save(_ context.Context, beer *domain.Beer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.beers[beer.ID().String()] = beerRecord{
		id:          beer.ID(),
		name:        beer.Name(),
		style:       beer.Style(),
		abv:         beer.ABV(),
		price:       beer.Price(),
		description: beer.Description(),
		status:      beer.Status(),
	}
	return nil
}

func (r *MemoryBeerRepository) FindByID(_ context.Context, id domain.BeerID) (*domain.Beer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.beers[id.String()]
	if !ok {
		return nil, domain.ErrBeerNotFound
	}
	return record.toEntity(), nil
}

func (r *MemoryBeerRepository) FindAll(_ context.Context) ([]*domain.Beer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	beers := make([]*domain.Beer, 0, len(r.beers))
	for _, record := range r.beers {
		beers = append(beers, record.toEntity())
	}
	sort.Slice(beers, func(i, j int) bool { return beers[i].Name() < beers[j].Name() })
	return beers, nil
}

func (r *MemoryBeerRepository) ExistsByName(_ context.Context, name string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(name))
	for _, record := range r.beers {
		if strings.ToLower(record.name) == needle {
			return true, nil
		}
	}
	return false, nil
}

func (record beerRecord) toEntity() *domain.Beer {
	return domain.Rehydrate(
		record.id,
		record.name,
		record.style,
		record.abv,
		record.price,
		record.description,
		record.status,
	)
}
