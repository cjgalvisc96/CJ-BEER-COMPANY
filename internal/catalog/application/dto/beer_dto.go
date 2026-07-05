// Package dto defines the data shapes crossing the catalog application
// boundary. Presentation depends on these, never on domain types.
package dto

import "github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"

type BeerOutput struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Style       string  `json:"style"`
	ABV         float64 `json:"abv"`
	PriceCents  int64   `json:"price_cents"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
}

func BeerOutputFromEntity(beer *domain.Beer) BeerOutput {
	return BeerOutput{
		ID:          beer.ID().String(),
		Name:        beer.Name(),
		Style:       beer.Style().String(),
		ABV:         beer.ABV().Value(),
		PriceCents:  beer.Price().Cents(),
		Currency:    beer.Price().Currency(),
		Description: beer.Description(),
		Status:      string(beer.Status()),
	}
}
