package eventhandlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/events"
)

type failingWriter struct{ err error }

func (w *failingWriter) CreateSalesOrder(context.Context, dtos.SalesOrder) error {
	return w.err
}

func TestProjectionSurfacesWriterFailure(t *testing.T) {
	writeErr := errors.New("projection store down")
	handler := NewSalesOrderCreatedEventHandler(&failingWriter{err: writeErr}, slog.Default())

	err := handler.Handle(context.Background(), events.NewSalesOrderCreated(
		sharedkernel.NewSalesOrderId(),
		uuid.New(),
		sharedkernel.SalesOrderNumber{Value: "20240315-1500"},
		sharedkernel.OrderDate{},
		sharedkernel.CustomerId{Value: uuid.New()},
		sharedkernel.CustomerName{Value: "Muflone"},
		nil,
	))

	assert.ErrorIs(t, err, writeErr)
}
