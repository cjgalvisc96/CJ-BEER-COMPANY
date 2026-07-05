// Package eventhandlers holds the Warehouses read-model projections —
// BrewUp.Warehouses.ReadModel/EventHandlers.
package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

// availabilityWriter is the slice of the read-model service this
// projection needs (the book's IAvailabilityService).
type availabilityWriter interface {
	UpsertAvailability(ctx context.Context, availability dtos.Availability) error
}

// AvailabilityProjector projects both availability events into the read
// model; each event carries the new cumulative quantity, so projecting is
// a plain upsert.
type AvailabilityProjector struct {
	service availabilityWriter
	logger  *slog.Logger
}

func NewAvailabilityProjector(service availabilityWriter, logger *slog.Logger) *AvailabilityProjector {
	return &AvailabilityProjector{service: service, logger: logger}
}

func (p *AvailabilityProjector) OnAvailabilityUpdatedDueToProductionOrder(
	ctx context.Context,
	event events.AvailabilityUpdatedDueToProductionOrder,
) error {
	return p.upsert(ctx, event.BeerId.Value.String(), event.BeerName.Value, event.Quantity)
}

func (p *AvailabilityProjector) OnBeerAvailabilityUpdated(
	ctx context.Context,
	event events.BeerAvailabilityUpdated,
) error {
	return p.upsert(ctx, event.BeerId.Value.String(), event.BeerName.Value, event.Quantity)
}

func (p *AvailabilityProjector) OnAvailabilityCompensated(
	ctx context.Context,
	event events.AvailabilityCompensated,
) error {
	return p.upsert(ctx, event.BeerId.Value.String(), event.BeerName.Value, event.Quantity)
}

func (p *AvailabilityProjector) upsert(
	ctx context.Context,
	beerId, beerName string,
	quantity customtypes.Quantity,
) error {
	if err := p.service.UpsertAvailability(ctx, dtos.Availability{
		BeerId:   beerId,
		BeerName: beerName,
		Quantity: quantity,
	}); err != nil {
		return err
	}
	p.logger.Info("warehouses.readmodel.availability_projected",
		slog.String("beer_id", beerId), slog.String("quantity", quantity.String()))
	return nil
}
