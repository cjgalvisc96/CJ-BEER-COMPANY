// Error-path specifications: violated invariants and infrastructure
// failures must commit nothing.
package commandhandlers_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain/commandhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

func TestProductionOrderWithNonPositiveQuantityFailsOnCreation(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToProductionOrder]{
		StreamName: domain.StreamName,
		Given:      func() []muflone.DomainEvent { return nil },
		When: func() commands.UpdateAvailabilityDueToProductionOrder {
			return commands.NewUpdateAvailabilityDueToProductionOrder(
				beerId, uuid.New(), sharedkernel.BeerName{Value: "BrewUp IPA"},
				customtypes.NewQuantity(0, "Lt"),
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToProductionOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrInvalidQuantity,
	}.Run(t)
}

func TestProductionOrderWithNonPositiveQuantityFailsOnUpdate(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "BrewUp IPA"}
	correlationId := uuid.New()

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToProductionOrder]{
		StreamName: domain.StreamName,
		Given: func() []muflone.DomainEvent {
			return []muflone.DomainEvent{
				events.NewAvailabilityUpdatedDueToProductionOrder(
					beerId, correlationId, beerName, customtypes.NewQuantity(100, "Lt"),
				),
			}
		},
		When: func() commands.UpdateAvailabilityDueToProductionOrder {
			return commands.NewUpdateAvailabilityDueToProductionOrder(
				beerId, correlationId, beerName, customtypes.NewQuantity(-5, "Lt"),
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToProductionOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrInvalidQuantity,
	}.Run(t)
}

func TestSalesAllocationForUnknownBeerFails(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}

	muflone.CommandSpecification[commands.UpdateAvailabilityDueToSalesOrder]{
		StreamName: domain.StreamName,
		Given:      func() []muflone.DomainEvent { return nil },
		When: func() commands.UpdateAvailabilityDueToSalesOrder {
			return commands.NewUpdateAvailabilityDueToSalesOrder(
				beerId, uuid.New(), sharedkernel.BeerName{Value: "Ghost Beer"},
				customtypes.NewQuantity(10, "Lt"),
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToSalesOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: muflone.ErrAggregateNotFound,
	}.Run(t)
}

func TestSalesAllocationWithNonPositiveQuantityFails(t *testing.T) {
	beerId := sharedkernel.BeerId{Value: uuid.New()}
	beerName := sharedkernel.BeerName{Value: "BrewUp IPA"}
	correlationId := uuid.New()

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
				beerId, correlationId, beerName, customtypes.NewQuantity(0, "Lt"),
			)
		},
		OnHandler: func(store muflone.EventStore) muflone.CommandHandler[commands.UpdateAvailabilityDueToSalesOrder] {
			return commandhandlers.NewUpdateAvailabilityDueToSalesOrderCommandHandler(
				newAvailabilityRepository(store), slog.Default(),
			)
		},
		Expect:        func() []muflone.DomainEvent { return nil },
		ExpectedError: domain.ErrInvalidQuantity,
	}.Run(t)
}

// failingRepository drives the infrastructure-error branch of the
// production-order handler (a store failure that is NOT "not found").
type failingRepository struct{ err error }

func (r *failingRepository) GetByID(context.Context, uuid.UUID) (*domain.Availability, error) {
	return nil, r.err
}

func (r *failingRepository) Save(context.Context, *domain.Availability, uuid.UUID) error {
	return r.err
}

func TestProductionOrderSurfacesStoreFailures(t *testing.T) {
	storeErr := errors.New("event store down")
	handler := commandhandlers.NewUpdateAvailabilityDueToProductionOrderCommandHandler(
		&failingRepository{err: storeErr}, slog.Default(),
	)

	err := handler.Handle(context.Background(), commands.NewUpdateAvailabilityDueToProductionOrder(
		sharedkernel.BeerId{Value: uuid.New()}, uuid.New(),
		sharedkernel.BeerName{Value: "BrewUp IPA"}, customtypes.NewQuantity(10, "Lt"),
	))

	assert.ErrorIs(t, err, storeErr)
}
