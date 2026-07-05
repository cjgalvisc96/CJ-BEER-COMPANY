// Specification tests for the Availability aggregate — the Go rendition of
// the book's Example 2 ("Updating Availability").
package commandhandlers_test

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

func newAvailabilityRepository(store muflone.EventStore) muflone.Repository[*domain.Availability] {
	return muflone.NewEventStoreRepository(store, domain.NewAvailability, domain.StreamName, nil)
}

// TestUpdateAvailabilityDueToProductionOrderAfterAggregateCreation mirrors
// the book's test one to one: Given the availability was set to 100 Lt by
// a prior production order, When another production order adds 100 Lt,
// Expect AvailabilityUpdatedDueToProductionOrder with the new total of
// 200 Lt.
func TestUpdateAvailabilityDueToProductionOrderAfterAggregateCreation(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	quantity := customtypes.NewQuantity(100, "Lt")
	newQuantity := customtypes.NewQuantity(200, "Lt")
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToProductionOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(beerId, correlationId, beerName, quantity),
			}
		},
		When: func() commands.UpdateAvailabilityDueToProductionOrder {
			return commands.NewUpdateAvailabilityDueToProductionOrder(beerId, correlationId, beerName, quantity)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToProductionOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(beerId, correlationId, beerName, newQuantity),
			}
		},
	}.Run(t)
}

// TestFirstProductionOrderCreatesAvailability: with no prior history the
// handler creates the aggregate and the event carries the produced
// quantity as the total.
func TestFirstProductionOrderCreatesAvailability(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	quantity := customtypes.NewQuantity(100, "Lt")
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToProductionOrder]{
		StreamName: domain.StreamName,
		Given:      func() []muflone.DomainEvent { return nil },
		When: func() commands.UpdateAvailabilityDueToProductionOrder {
			return commands.NewUpdateAvailabilityDueToProductionOrder(beerId, correlationId, beerName, quantity)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToProductionOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(beerId, correlationId, beerName, quantity),
			}
		},
	}.Run(t)
}

// TestAllocationForSalesOrderEmitsBeerAvailabilityUpdated: allocating 30
// out of 100 leaves 70 — the BeerAvailabilityUpdated event carries the
// remaining availability and the saga correlation.
func TestAllocationForSalesOrderEmitsBeerAvailabilityUpdated(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	correlationId := uuid.New()
	salesOrderId := uuid.NewString()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"),
				),
			}
		},
		When: func() commands.UpdateAvailabilityDueToSalesOrder {
			return commands.NewUpdateAvailabilityDueToSalesOrder(
				beerId, correlationId, beerName, customtypes.NewQuantity(30, "Lt"), salesOrderId,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToSalesOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewBeerAvailabilityUpdated(
					beerId, correlationId, beerName, customtypes.NewQuantity(70, "Lt"), salesOrderId,
				),
			}
		},
	}.Run(t)
}

// TestAllocationBeyondAvailabilityRecordsQuantityNotFound: the shortage is
// the book's Ch. 12 failure event — recorded, not silently dropped — so
// the saga can compensate.
func TestAllocationBeyondAvailabilityRecordsQuantityNotFound(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	correlationId := uuid.New()
	salesOrderId := uuid.NewString()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(10, "Lt"),
				),
			}
		},
		When: func() commands.UpdateAvailabilityDueToSalesOrder {
			return commands.NewUpdateAvailabilityDueToSalesOrder(
				beerId, correlationId, beerName, customtypes.NewQuantity(999, "Lt"), salesOrderId,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToSalesOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewQuantityNotFound(
					beerId, correlationId, salesOrderId,
					customtypes.NewQuantity(999, "Lt"), customtypes.NewQuantity(10, "Lt"),
				),
			}
		},
	}.Run(t)
}

// TestRedrivenAllocationIsIdempotent (book Ch. 12: retried steps must not
// produce side effects): the order already holds stock, so the re-driven
// command re-emits the fact with the quantity UNCHANGED.
func TestRedrivenAllocationIsIdempotent(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	correlationId := uuid.New()
	salesOrderId := uuid.NewString()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToSalesOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"),
				),
				events.NewBeerAvailabilityUpdated(
					beerId, correlationId, beerName, customtypes.NewQuantity(70, "Lt"), salesOrderId,
				),
			}
		},
		When: func() commands.UpdateAvailabilityDueToSalesOrder {
			return commands.NewUpdateAvailabilityDueToSalesOrder(
				beerId, correlationId, beerName, customtypes.NewQuantity(30, "Lt"), salesOrderId,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToSalesOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewBeerAvailabilityUpdated(
					beerId, correlationId, beerName, customtypes.NewQuantity(70, "Lt"), salesOrderId,
				),
			}
		},
	}.Run(t)
}

// TestRedrivenCompensationIsIdempotent: the stock was already given back —
// the duplicate compensation re-emits the fact with the total UNCHANGED.
func TestRedrivenCompensationIsIdempotent(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	correlationId := uuid.New()
	salesOrderId := uuid.NewString()

	muflone.CommandSpecification[commands.CompensateAvailabilityDueToFailedAllocation]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"),
				),
				events.NewBeerAvailabilityUpdated(
					beerId, correlationId, beerName, customtypes.NewQuantity(70, "Lt"), salesOrderId,
				),
				events.NewAvailabilityCompensated(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"), salesOrderId,
				),
			}
		},
		When: func() commands.CompensateAvailabilityDueToFailedAllocation {
			return commands.NewCompensateAvailabilityDueToFailedAllocation(
				beerId, correlationId, customtypes.NewQuantity(30, "Lt"), salesOrderId,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.CompensateAvailabilityDueToFailedAllocation] {
			return commandhandlers.NewCompensateAvailabilityDueToFailedAllocationCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityCompensated(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"), salesOrderId,
				),
			}
		},
	}.Run(t)
}

// TestCompensationRestoresAllocatedQuantity: the compensating transaction
// (book Ch. 12, backward recovery) gives allocated stock back.
func TestCompensationRestoresAllocatedQuantity(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "Muflone IPA"}
	correlationId := uuid.New()
	salesOrderId := uuid.NewString()

	muflone.CommandSpecification[commands.CompensateAvailabilityDueToFailedAllocation]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"),
				),
				events.NewBeerAvailabilityUpdated(
					beerId, correlationId, beerName, customtypes.NewQuantity(70, "Lt"), salesOrderId,
				),
			}
		},
		When: func() commands.CompensateAvailabilityDueToFailedAllocation {
			return commands.NewCompensateAvailabilityDueToFailedAllocation(
				beerId, correlationId, customtypes.NewQuantity(30, "Lt"), salesOrderId,
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.CompensateAvailabilityDueToFailedAllocation] {
			return commandhandlers.NewCompensateAvailabilityDueToFailedAllocationCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityCompensated(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"), salesOrderId,
				),
			}
		},
	}.Run(t)
}
