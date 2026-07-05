// Package commands holds the Warehouses module's commands —
// BrewUp.Warehouses.SharedKernel/Commands.
package commands

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
)

// UpdateAvailabilityDueToProductionOrder mirrors the book's command: a
// production order finished, add the produced quantity to the beer's
// availability. The aggregateId is the BeerId.
type UpdateAvailabilityDueToProductionOrder struct {
	muflone.CommandBase
	BeerId   sharedkernel.BeerId   `json:"beer_id"`
	BeerName sharedkernel.BeerName `json:"beer_name"`
	Quantity customtypes.Quantity  `json:"quantity"`
}

func NewUpdateAvailabilityDueToProductionOrder(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) UpdateAvailabilityDueToProductionOrder {
	return UpdateAvailabilityDueToProductionOrder{
		CommandBase: muflone.NewCommandBase(beerId.Value, commitId),
		BeerId:      beerId,
		BeerName:    beerName,
		Quantity:    quantity,
	}
}

func (UpdateAvailabilityDueToProductionOrder) MessageName() string {
	return "warehouses.update_availability_due_to_production_order"
}

// UpdateAvailabilityDueToSalesOrder allocates stock for a sales order:
// subtract the ordered quantity from the beer's availability.
type UpdateAvailabilityDueToSalesOrder struct {
	muflone.CommandBase
	BeerId   sharedkernel.BeerId   `json:"beer_id"`
	BeerName sharedkernel.BeerName `json:"beer_name"`
	Quantity customtypes.Quantity  `json:"quantity"`
}

func NewUpdateAvailabilityDueToSalesOrder(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) UpdateAvailabilityDueToSalesOrder {
	return UpdateAvailabilityDueToSalesOrder{
		CommandBase: muflone.NewCommandBase(beerId.Value, commitId),
		BeerId:      beerId,
		BeerName:    beerName,
		Quantity:    quantity,
	}
}

func (UpdateAvailabilityDueToSalesOrder) MessageName() string {
	return "warehouses.update_availability_due_to_sales_order"
}
