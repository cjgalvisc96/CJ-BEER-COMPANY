// Package ports declares the driven ports the orders use cases need from
// other contexts. The adapters live in orders/infrastructure — this is the
// anti-corruption layer keeping catalog types out of the orders context.
package ports

import (
	"context"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// BeerSnapshot is the orders-local view of a catalog beer: only what
// placing an order needs.
type BeerSnapshot struct {
	ID       string
	Name     string
	Price    shared.Money
	Sellable bool
}

// BeerCatalog looks up beers in the catalog context.
type BeerCatalog interface {
	// FindBeer returns the snapshot, or a not-found domain error.
	FindBeer(ctx context.Context, beerID string) (BeerSnapshot, error)
}
