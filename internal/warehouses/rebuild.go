package warehouses

import (
	"context"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

// RebuildableStore is what a projection rebuild needs from the event
// store: enumerate the streams and replay them.
type RebuildableStore interface {
	muflone.EventStore
	muflone.StreamLister
}

// RebuildReadModel replays every Availability stream through the module's
// projector. Saga streams are skipped: the saga IS its stream, it has no
// projection. The caller resets the target first.
func RebuildReadModel(ctx context.Context, store RebuildableStore, readModel availabilityReadModel, logger *slog.Logger) error {
	projector := eventhandlers.NewAvailabilityProjector(readModel, logger)

	streams, err := store.ListStreams(ctx, domain.StreamName)
	if err != nil {
		return err
	}
	replayed := 0
	for _, streamID := range streams {
		stored, err := store.ReadStream(ctx, streamID)
		if err != nil {
			return err
		}
		for _, record := range stored {
			var err error
			switch event := record.Event.(type) {
			case events.AvailabilityUpdatedDueToProductionOrder:
				err = projector.OnAvailabilityUpdatedDueToProductionOrder(ctx, event)
			case events.BeerAvailabilityUpdated:
				err = projector.OnBeerAvailabilityUpdated(ctx, event)
			case events.AvailabilityCompensated:
				err = projector.OnAvailabilityCompensated(ctx, event)
			}
			if err != nil {
				return err
			}
			replayed++
		}
	}
	logger.Info("warehouses.readmodel.rebuilt",
		slog.Int("streams", len(streams)), slog.Int("events", replayed))
	return nil
}
