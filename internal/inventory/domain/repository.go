package domain

import "context"

type StockRepository interface {
	Save(ctx context.Context, item *StockItem) error
	FindByBeerID(ctx context.Context, beerID BeerRef) (*StockItem, error)
	FindAll(ctx context.Context) ([]*StockItem, error)
	Exists(ctx context.Context, beerID BeerRef) (bool, error)
}
