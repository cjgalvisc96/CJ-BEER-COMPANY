// Package ports declares the driven ports the brewing use cases need from
// other contexts.
package ports

import "context"

// BeerCatalog answers the single question brewing has for the catalog:
// can this beer be brewed right now?
type BeerCatalog interface {
	IsBrewable(ctx context.Context, beerID string) (bool, error)
}
