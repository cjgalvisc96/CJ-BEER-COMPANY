// Package acl adapts the catalog context to the orders BeerCatalog port.
// It is the only place in the orders context allowed to import catalog
// code, and it talks to catalog's application layer (its public API inside
// the monolith), never its domain.
package acl

import (
	"context"

	catalogqueries "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/ports"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type CatalogAdapter struct {
	getBeer *catalogqueries.GetBeerHandler
}

func NewCatalogAdapter(getBeer *catalogqueries.GetBeerHandler) *CatalogAdapter {
	return &CatalogAdapter{getBeer: getBeer}
}

func (a *CatalogAdapter) FindBeer(ctx context.Context, beerID string) (ports.BeerSnapshot, error) {
	beer, err := a.getBeer.Handle(ctx, beerID)
	if err != nil {
		return ports.BeerSnapshot{}, err
	}
	price, err := shared.NewMoney(beer.PriceCents, beer.Currency)
	if err != nil {
		return ports.BeerSnapshot{}, err
	}
	return ports.BeerSnapshot{
		ID:       beer.ID,
		Name:     beer.Name,
		Price:    price,
		Sellable: beer.Status == "active",
	}, nil
}
