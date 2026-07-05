package eventhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/sharedkernel/events"
)

type failingWriter struct{ err error }

func (w *failingWriter) UpsertAvailability(context.Context, dtos.Availability) error {
	return w.err
}

func TestProjectorSurfacesWriterFailure(t *testing.T) {
	writeErr := errors.New("projection store down")
	projector := NewAvailabilityProjector(&failingWriter{err: writeErr}, slog.Default())

	err := projector.OnBeerAvailabilityUpdated(context.Background(), events.NewBeerAvailabilityUpdated(
		sharedkernel.BeerId{Value: uuid.New()},
		uuid.New(),
		sharedkernel.BeerName{Value: "BrewUp IPA"},
		customtypes.NewQuantity(70, "Lt"),
	))

	assert.ErrorIs(t, err, writeErr)
}
