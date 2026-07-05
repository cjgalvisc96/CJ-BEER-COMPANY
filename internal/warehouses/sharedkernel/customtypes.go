// Package sharedkernel is the published language of the Warehouses module
// — BrewUp.Warehouses.SharedKernel.
package sharedkernel

import "github.com/google/uuid"

type BeerId struct {
	Value uuid.UUID `json:"value"`
}

type BeerName struct {
	Value string `json:"value"`
}
