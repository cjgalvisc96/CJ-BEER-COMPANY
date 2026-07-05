package integration_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/integration"
)

func TestHandlerRejectsMalformedPayload(t *testing.T) {
	handler := integration.NewSalesOrderCreatedHandler(muflone.NewServiceBus(slog.Default()), slog.Default())

	err := handler.Handle(context.Background(), []byte(`not json`))

	assert.Error(t, err)
}

func TestHandlerSkipsRowsWithInvalidBeerId(t *testing.T) {
	handler := integration.NewSalesOrderCreatedHandler(muflone.NewServiceBus(slog.Default()), slog.Default())

	err := handler.Handle(context.Background(), []byte(`{
		"commit_id": "`+uuid.NewString()+`",
		"sales_order_id": "`+uuid.NewString()+`",
		"rows": [{"beer_id": "not-a-uuid", "beer_name": "BrewUp IPA", "quantity": 10, "unit_of_measure": "Lt"}]
	}`))

	assert.NoError(t, err, "an invalid row is skipped, not a poison message")
}

func TestHandlerSurfacesBusFailures(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	handler := integration.NewSalesOrderCreatedHandler(bus, slog.Default())

	err := handler.Handle(context.Background(), []byte(`{
		"commit_id": "`+uuid.NewString()+`",
		"sales_order_id": "`+uuid.NewString()+`",
		"rows": [{"beer_id": "`+uuid.NewString()+`", "beer_name": "BrewUp IPA", "quantity": 10, "unit_of_measure": "Lt"}]
	}`))

	assert.Error(t, err)
}
