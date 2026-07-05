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

// UpdateAvailabilityDueToSalesOrder is one step of the order-allocation
// saga: allocate the ordered quantity of one beer. SalesOrderId correlates
// the step with its saga.
type UpdateAvailabilityDueToSalesOrder struct {
	muflone.CommandBase
	BeerId       sharedkernel.BeerId   `json:"beer_id"`
	BeerName     sharedkernel.BeerName `json:"beer_name"`
	Quantity     customtypes.Quantity  `json:"quantity"`
	SalesOrderId string                `json:"sales_order_id"`
}

func NewUpdateAvailabilityDueToSalesOrder(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
	salesOrderId string,
) UpdateAvailabilityDueToSalesOrder {
	return UpdateAvailabilityDueToSalesOrder{
		CommandBase:  muflone.NewCommandBase(beerId.Value, commitId),
		BeerId:       beerId,
		BeerName:     beerName,
		Quantity:     quantity,
		SalesOrderId: salesOrderId,
	}
}

func (UpdateAvailabilityDueToSalesOrder) MessageName() string {
	return "warehouses.update_availability_due_to_sales_order"
}

// CompensateAvailabilityDueToFailedAllocation is the saga's compensating
// transaction (book Ch. 12, backward recovery): give an already-allocated
// quantity back because a later step of the same order failed.
type CompensateAvailabilityDueToFailedAllocation struct {
	muflone.CommandBase
	BeerId       sharedkernel.BeerId  `json:"beer_id"`
	Quantity     customtypes.Quantity `json:"quantity"`
	SalesOrderId string               `json:"sales_order_id"`
}

func NewCompensateAvailabilityDueToFailedAllocation(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	quantity customtypes.Quantity,
	salesOrderId string,
) CompensateAvailabilityDueToFailedAllocation {
	return CompensateAvailabilityDueToFailedAllocation{
		CommandBase:  muflone.NewCommandBase(beerId.Value, commitId),
		BeerId:       beerId,
		Quantity:     quantity,
		SalesOrderId: salesOrderId,
	}
}

func (CompensateAvailabilityDueToFailedAllocation) MessageName() string {
	return "warehouses.compensate_availability_due_to_failed_allocation"
}
