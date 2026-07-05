// Package domain holds the Warehouses write model: the event-sourced
// Availability aggregate (one per beer) and the event-sourced
// OrderAllocationSaga — BrewUp.Warehouses.Domain.
package domain

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

const StreamName = "Availability"

var ErrInvalidQuantity = muflone.ErrInvalid("quantity must be positive")

// Availability tracks how much of one beer the warehouse can sell. Its
// events carry the NEW cumulative quantity, matching the book's
// specification test (100 Lt given + 100 Lt produced → event with 200 Lt).
type Availability struct {
	muflone.AggregateRoot

	beerId   sharedkernel.BeerId
	beerName sharedkernel.BeerName
	quantity customtypes.Quantity
}

// NewAvailability returns an empty aggregate for stream replay.
func NewAvailability() *Availability {
	availability := &Availability{}
	availability.Bind(availability)
	return availability
}

// CreateAvailability starts tracking a beer with its first production
// output.
func CreateAvailability(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) (*Availability, error) {
	if quantity.Value <= 0 {
		return nil, ErrInvalidQuantity
	}
	availability := NewAvailability()
	availability.RaiseEvent(events.NewAvailabilityUpdatedDueToProductionOrder(
		beerId, commitId, beerName, quantity,
	))
	return availability, nil
}

// UpdateDueToProductionOrder adds produced units; the raised event carries
// the new total.
func (a *Availability) UpdateDueToProductionOrder(
	commitId uuid.UUID,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) error {
	if quantity.Value <= 0 {
		return ErrInvalidQuantity
	}
	newTotal := customtypes.NewQuantity(a.quantity.Value+quantity.Value, quantity.UnitOfMeasure)
	a.RaiseEvent(events.NewAvailabilityUpdatedDueToProductionOrder(
		a.beerId, commitId, beerName, newTotal,
	))
	return nil
}

// UpdateDueToSalesOrder is one saga step: allocate stock to a sales order.
// A shortage is not an error — it is the book's QuantityNotFound event
// (Ch. 12, Figure 12.3), recorded in the stream so the saga can react.
func (a *Availability) UpdateDueToSalesOrder(
	commitId uuid.UUID,
	quantity customtypes.Quantity,
	salesOrderId string,
) error {
	if quantity.Value <= 0 {
		return ErrInvalidQuantity
	}
	if quantity.Value > a.quantity.Value {
		a.RaiseEvent(events.NewQuantityNotFound(
			a.beerId, commitId, salesOrderId, quantity, a.quantity,
		))
		return nil
	}
	remaining := customtypes.NewQuantity(a.quantity.Value-quantity.Value, a.quantity.UnitOfMeasure)
	a.RaiseEvent(events.NewBeerAvailabilityUpdated(a.beerId, commitId, a.beerName, remaining, salesOrderId))
	return nil
}

// RefuseUnknownBeer records QuantityNotFound for a beer the warehouse has
// never stocked: available is zero by definition, and the refusal must
// still land in the beer's stream so the saga can react.
func RefuseUnknownBeer(
	beerId sharedkernel.BeerId,
	commitId uuid.UUID,
	requested customtypes.Quantity,
	salesOrderId string,
) *Availability {
	availability := NewAvailability()
	availability.RaiseEvent(events.NewQuantityNotFound(
		beerId, commitId, salesOrderId, requested,
		customtypes.NewQuantity(0, requested.UnitOfMeasure),
	))
	return availability
}

// CompensateDueToFailedAllocation is the compensating transaction (book
// Ch. 12, backward recovery): give an allocated quantity back.
func (a *Availability) CompensateDueToFailedAllocation(
	commitId uuid.UUID,
	quantity customtypes.Quantity,
	salesOrderId string,
) error {
	if quantity.Value <= 0 {
		return ErrInvalidQuantity
	}
	newTotal := customtypes.NewQuantity(a.quantity.Value+quantity.Value, a.quantity.UnitOfMeasure)
	a.RaiseEvent(events.NewAvailabilityCompensated(a.beerId, commitId, a.beerName, newTotal, salesOrderId))
	return nil
}

// Route dispatches events to the apply methods.
func (a *Availability) Route(event muflone.DomainEvent) {
	switch e := event.(type) {
	case events.AvailabilityUpdatedDueToProductionOrder:
		a.applyAvailabilityUpdated(e.BeerId, e.BeerName, e.Quantity)
	case events.BeerAvailabilityUpdated:
		a.applyAvailabilityUpdated(e.BeerId, e.BeerName, e.Quantity)
	case events.AvailabilityCompensated:
		a.applyAvailabilityUpdated(e.BeerId, e.BeerName, e.Quantity)
	case events.QuantityNotFound:
		// A refusal changes no quantity; it only anchors the stream id.
		s := e.BeerId
		a.SetID(s.Value)
		a.beerId = s
	}
}

func (a *Availability) applyAvailabilityUpdated(
	beerId sharedkernel.BeerId,
	beerName sharedkernel.BeerName,
	quantity customtypes.Quantity,
) {
	a.SetID(beerId.Value)
	a.beerId = beerId
	a.beerName = beerName
	a.quantity = quantity
}
