package domain

import "context"

// BeerRepository is the persistence port of the catalog aggregate. Defined
// in the domain, implemented in infrastructure (dependency inversion).
type BeerRepository interface {
	Save(ctx context.Context, beer *Beer) error
	FindByID(ctx context.Context, id BeerID) (*Beer, error)
	FindAll(ctx context.Context) ([]*Beer, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
}
