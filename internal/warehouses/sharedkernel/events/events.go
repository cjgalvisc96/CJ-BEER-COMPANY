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
// SalesOrderId correlates the allocation with its saga (added per weak
// schema — ADR-0007 — old stored events simply decode with it empty).
type BeerAvailabilityUpdated struct {
	muflone.DomainEventBase
	BeerId       sharedkernel.BeerId   `json:"beer_id"`
	BeerName     sharedkernel.BeerName `json:"beer_name"`
	Quantity     customtypes.Quantity  `json:"quantity"`
	SalesOrderId string                `json:"sales_order_id"`
}

func NewBeerAvailabilityUpdated(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
	salesOrderId string,
) BeerAvailabilityUpdated {
	return BeerAvailabilityUpdated{
		DomainEventBase: muflone.NewDomainEventBase(beerId.Value, commitId),
		BeerId:          beerId,
		BeerName:        beerName,
		Quantity:        quantity,
		SalesOrderId:    salesOrderId,
	}
}

func (BeerAvailabilityUpdated) MessageName() string {
	return "warehouses.beer_availability_updated"
}

// QuantityNotFound is the book's failure event (Chapter 12, Figure 12.3):
// the warehouse cannot fulfil an allocation because the available stock is
// insufficient. It is a recorded fact — the refusal is part of the beer's
// history.
type QuantityNotFound struct {
	muflone.DomainEventBase
	BeerId       sharedkernel.BeerId  `json:"beer_id"`
	SalesOrderId string               `json:"sales_order_id"`
	Requested    customtypes.Quantity `json:"requested"`
	Available    customtypes.Quantity `json:"available"`
}

func NewQuantityNotFound(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	salesOrderId string,
	requested customtypes.Quantity,
	available customtypes.Quantity,
) QuantityNotFound {
	return QuantityNotFound{
		DomainEventBase: muflone.NewDomainEventBase(beerId.Value, commitId),
		BeerId:          beerId,
		SalesOrderId:    salesOrderId,
		Requested:       requested,
		Available:       available,
	}
}

func (QuantityNotFound) MessageName() string {
	return "warehouses.quantity_not_found"
}

// AvailabilityCompensated is the compensating transaction's fact: a
// previously allocated quantity was given back because the order's saga
// failed. Quantity is the new cumulative availability.
type AvailabilityCompensated struct {
	muflone.DomainEventBase
	BeerId       sharedkernel.BeerId   `json:"beer_id"`
	BeerName     sharedkernel.BeerName `json:"beer_name"`
	Quantity     customtypes.Quantity  `json:"quantity"`
	SalesOrderId string                `json:"sales_order_id"`
}

func NewAvailabilityCompensated(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
	salesOrderId string,
) AvailabilityCompensated {
	return AvailabilityCompensated{
		DomainEventBase: muflone.NewDomainEventBase(beerId.Value, commitId),
		BeerId:          beerId,
		BeerName:        beerName,
		Quantity:        quantity,
		SalesOrderId:    salesOrderId,
	}
}

func (AvailabilityCompensated) MessageName() string {
	return "warehouses.availability_compensated"
}
