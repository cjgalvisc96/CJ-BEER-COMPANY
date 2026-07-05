// Package dto defines the data shapes crossing the inventory application
// boundary.
package dto

import "github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"

type StockOutput struct {
	BeerID       string `json:"beer_id"`
	Quantity     int    `json:"quantity"`
	ReorderLevel int    `json:"reorder_level"`
}

func StockOutputFromEntity(item *domain.StockItem) StockOutput {
	return StockOutput{
		BeerID:       item.BeerID().String(),
		Quantity:     item.Quantity(),
		ReorderLevel: item.ReorderLevel(),
	}
}
