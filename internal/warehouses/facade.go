// Package warehouses is the Warehouses bounded context of CJ Beer Company
// — the Go rendition of the book's BrewUp Warehouses module.
package warehouses

import (
	"context"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/commands"
)

// ProductionOrderJson is the inbound payload declaring a finished
// production order for a beer.
type ProductionOrderJson struct {
	BeerId   string               `json:"beer_id" binding:"required"`
	BeerName string               `json:"beer_name" binding:"required"`
	Quantity customtypes.Quantity `json:"quantity"`
}

// AvailabilityQueries is the read-side port of the facade; the in-memory
// and Postgres read-model services both implement it.
type AvailabilityQueries interface {
	GetAvailability(ctx context.Context, beerId string) (dtos.Availability, error)
	GetAvailabilities(ctx context.Context) ([]dtos.Availability, error)
}

// Facade is the module's public surface (the book's IWarehousesFacade).
type Facade struct {
	bus     *muflone.ServiceBus
	queries AvailabilityQueries
}

func NewFacade(bus *muflone.ServiceBus, queries AvailabilityQueries) *Facade {
	return &Facade{bus: bus, queries: queries}
}

// UpdateAvailabilityDueToProductionOrder sends the command that adds the
// produced quantity to the beer's availability.
func (f *Facade) UpdateAvailabilityDueToProductionOrder(ctx context.Context, body ProductionOrderJson) (string, error) {
	beerId, err := uuid.Parse(body.BeerId)
	if err != nil {
		return "", muflone.ErrInvalid("invalid beer id: " + body.BeerId)
	}
	command := commands.NewUpdateAvailabilityDueToProductionOrder(
		sharedkernel.BeerId{Value: beerId},
		uuid.New(),
		sharedkernel.BeerName{Value: body.BeerName},
		body.Quantity,
	)
	if err := f.bus.Send(ctx, command); err != nil {
		return "", err
	}
	return beerId.String(), nil
}

func (f *Facade) GetAvailability(ctx context.Context, beerId string) (dtos.Availability, error) {
	return f.queries.GetAvailability(ctx, beerId)
}

func (f *Facade) GetAvailabilities(ctx context.Context) ([]dtos.Availability, error) {
	return f.queries.GetAvailabilities(ctx)
}
