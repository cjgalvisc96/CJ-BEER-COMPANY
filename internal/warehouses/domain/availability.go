// Package domain holds the Warehouses write model: the event-sourced
// Availability aggregate (one per beer) — BrewUp.Warehouses.Domain.
package domain

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

const StreamName = "Availability"

var (
	ErrInvalidQuantity = muflone.ErrInvalid("quantity must be positive")
	ErrNotEnoughStock  = muflone.ErrInvalid("not enough beer available")
)

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

// UpdateDueToSalesOrder allocates stock to a sales order; the raised
// BeerAvailabilityUpdated carries the remaining quantity.
func (a *Availability) UpdateDueToSalesOrder(commitId uuid.UUID, quantity customtypes.Quantity) error {
	if quantity.Value <= 0 {
		return ErrInvalidQuantity
	}
	if quantity.Value > a.quantity.Value {
		return fmt.Errorf("%w: %s has %d, requested %d",
			ErrNotEnoughStock, a.beerName.Value, a.quantity.Value, quantity.Value)
	}
	remaining := customtypes.NewQuantity(a.quantity.Value-quantity.Value, a.quantity.UnitOfMeasure)
	a.RaiseEvent(events.NewBeerAvailabilityUpdated(a.beerId, commitId, a.beerName, remaining))
	return nil
}

// Route dispatches events to the apply methods.
func (a *Availability) Route(event muflone.DomainEvent) {
	switch e := event.(type) {
	case events.AvailabilityUpdatedDueToProductionOrder:
		a.applyAvailabilityUpdated(e.BeerId, e.BeerName, e.Quantity)
	case events.BeerAvailabilityUpdated:
		a.applyAvailabilityUpdated(e.BeerId, e.BeerName, e.Quantity)
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
