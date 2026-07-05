// Package events holds the Warehouses module's domain events —
// BrewUp.Warehouses.SharedKernel/Events.
package events

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
)

// AvailabilityUpdatedDueToProductionOrder mirrors the book's event: the
// Quantity carries the NEW cumulative availability (the spec test in the
// book expects 200 Lt after 100 + a 100 production order).
type AvailabilityUpdatedDueToProductionOrder struct {
	muflone.DomainEventBase
	BeerId   sharedkernel.BeerId   `json:"beer_id"`
	BeerName sharedkernel.BeerName `json:"beer_name"`
	Quantity customtypes.Quantity  `json:"quantity"`
}

func NewAvailabilityUpdatedDueToProductionOrder(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) AvailabilityUpdatedDueToProductionOrder {
	return AvailabilityUpdatedDueToProductionOrder{
		DomainEventBase: muflone.NewDomainEventBase(beerId.Value, commitId),
		BeerId:          beerId,
		BeerName:        beerName,
		Quantity:        quantity,
	}
}

func (AvailabilityUpdatedDueToProductionOrder) MessageName() string {
	return "warehouses.availability_updated_due_to_production_order"
}

// BeerAvailabilityUpdated mirrors the book's
// BeerAvailabilityUpdated(BeerId, commitId, BeerName, Quantity): stock was
// allocated to a sales order, Quantity is the remaining availability.
type BeerAvailabilityUpdated struct {
	muflone.DomainEventBase
	BeerId   sharedkernel.BeerId   `json:"beer_id"`
	BeerName sharedkernel.BeerName `json:"beer_name"`
	Quantity customtypes.Quantity  `json:"quantity"`
}

func NewBeerAvailabilityUpdated(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) BeerAvailabilityUpdated {
	return BeerAvailabilityUpdated{
		DomainEventBase: muflone.NewDomainEventBase(beerId.Value, commitId),
		BeerId:          beerId,
		BeerName:        beerName,
		Quantity:        quantity,
	}
}

func (BeerAvailabilityUpdated) MessageName() string {
	return "warehouses.beer_availability_updated"
}
