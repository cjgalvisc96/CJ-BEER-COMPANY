package sales_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

func validOrder(beerId string) sales.SalesOrderJson {
	return sales.SalesOrderJson{
		SalesOrderNumber: "20240315-1500",
		CustomerName:     "Muflone",
		Rows: []sales.SalesOrderRowJson{{
			BeerId:   beerId,
			BeerName: "BrewUp IPA",
			Quantity: customtypes.NewQuantity(10, "Lt"),
			Price:    customtypes.NewPrice(5, "EUR"),
		}},
	}
}

func TestFacadeRejectsInvalidBeerId(t *testing.T) {
	facade := sales.NewFacade(muflone.NewServiceBus(slog.Default()), services.NewSalesOrderService())

	_, err := facade.CreateSalesOrder(context.Background(), validOrder("not-a-uuid"))

	var invalid muflone.ErrInvalid
	assert.ErrorAs(t, err, &invalid)
}

func TestFacadeSurfacesBusFailures(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	facade := sales.NewFacade(bus, services.NewSalesOrderService())

	_, err := facade.CreateSalesOrder(context.Background(), validOrder(uuid.NewString()))

	assert.Error(t, err)
}
