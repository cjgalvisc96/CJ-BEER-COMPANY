// Package acl adapts the catalog context to the brewing BeerCatalog port.
package acl

import (
	"context"

	catalogqueries "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type CatalogAdapter struct {
	getBeer *catalogqueries.GetBeerHandler
}

func NewCatalogAdapter(getBeer *catalogqueries.GetBeerHandler) *CatalogAdapter {
	return &CatalogAdapter{getBeer: getBeer}
}

func (a *CatalogAdapter) IsBrewable(ctx context.Context, beerID string) (bool, error) {
	beer, err := a.getBeer.Handle(ctx, beerID)
	if err != nil {
		// Unknown or malformed id → simply not brewable; anything else is
		// a real failure the caller should see.
		if _, isDomain := shared.KindOf(err); isDomain {
			return false, nil
		}
		return false, err
	}
	return beer.Status == "active", nil
}
