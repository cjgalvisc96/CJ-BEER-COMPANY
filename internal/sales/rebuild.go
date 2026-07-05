package sales

import (
	"context"
	"log/slog"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/eventhandlers"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

// RebuildableStore is what a projection rebuild needs from the event
// store: enumerate the streams and replay them.
type RebuildableStore interface {
	muflone.EventStore
	muflone.StreamLister
}

// RebuildReadModel replays every SalesOrder stream through the module's
// projections — the operational payoff of event sourcing: the read model
// can be reconstructed (or a brand-new projection populated) from the
// source of truth at any time. The caller resets the target first.
func RebuildReadModel(ctx context.Context, store RebuildableStore, readModel salesOrderReadModel, logger *slog.Logger) error {
	projection := eventhandlers.NewSalesOrderCreatedEventHandler(readModel, logger)
	statusProjector := eventhandlers.NewAllocationStatusProjector(readModel, logger)

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
			case events.SalesOrderCreated:
				err = projection.Handle(ctx, event)
			case events.SalesOrderAllocated:
				err = statusProjector.OnSalesOrderAllocated(ctx, event)
			case events.SalesOrderAllocationRejected:
				err = statusProjector.OnSalesOrderAllocationRejected(ctx, event)
			}
			if err != nil {
				return err
			}
			replayed++
		}
	}
	logger.Info("sales.readmodel.rebuilt",
		slog.Int("streams", len(streams)), slog.Int("events", replayed))
	return nil
}
