package eventhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

type failingStatusWriter struct{ err error }

func (w *failingStatusWriter) UpdateAllocationStatus(context.Context, string, string, string) error {
	return w.err
}

func TestStatusProjectionSurfacesWriterFailure(t *testing.T) {
	writeErr := errors.New("projection store down")
	projector := NewAllocationStatusProjector(&failingStatusWriter{err: writeErr}, slog.Default())
	salesOrderId := sharedkernel.NewSalesOrderId()

	assert.ErrorIs(t, projector.OnSalesOrderAllocated(context.Background(),
		events.NewSalesOrderAllocated(salesOrderId, uuid.New())), writeErr)
	assert.ErrorIs(t, projector.OnSalesOrderAllocationRejected(context.Background(),
		events.NewSalesOrderAllocationRejected(salesOrderId, uuid.New(), "shortage")), writeErr)
}
