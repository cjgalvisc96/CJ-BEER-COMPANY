// Package integrationevents holds the events the Warehouses module shares
// with other bounded contexts.
package integrationevents

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
)

const BeerAvailabilityUpdatedName = "warehouses.beer_availability_updated"

// BeerAvailabilityUpdated notifies other contexts (Sales) that stock was
// allocated and what remains.
type BeerAvailabilityUpdated struct {
	muflone.IntegrationEventBase
	BeerId        string `json:"beer_id"`
	BeerName      string `json:"beer_name"`
	Quantity      int    `json:"quantity"`
	UnitOfMeasure string `json:"unit_of_measure"`
}

func NewBeerAvailabilityUpdated(
	beerId uuid.UUID,
	commitId uuid.UUID,
	beerName string,
	quantity int,
	unitOfMeasure string,
) BeerAvailabilityUpdated {
	return BeerAvailabilityUpdated{
		IntegrationEventBase: muflone.NewIntegrationEventBase(commitId),
		BeerId:               beerId.String(),
		BeerName:             beerName,
		Quantity:             quantity,
		UnitOfMeasure:        unitOfMeasure,
	}
}

func (BeerAvailabilityUpdated) MessageName() string { return BeerAvailabilityUpdatedName }
