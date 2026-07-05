// Package dtos holds the Warehouses read-model DTOs —
// BrewUp.Warehouses.ReadModel/Dtos. Availability mirrors the book's DTO:
// beer id, beer name, and the quantity on hand.
package dtos

import "github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"

type Availability struct {
	BeerId   string               `json:"beer_id"`
	BeerName string               `json:"beer_name"`
	Quantity customtypes.Quantity `json:"quantity"`
}
