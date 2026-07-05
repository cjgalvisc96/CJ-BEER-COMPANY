package warehouses_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func TestFacadeRejectsInvalidBeerId(t *testing.T) {
	facade := warehouses.NewFacade(muflone.NewServiceBus(slog.Default()), services.NewAvailabilityService())

	_, err := facade.UpdateAvailabilityDueToProductionOrder(context.Background(), warehouses.ProductionOrderJson{
		BeerId:   "not-a-uuid",
		BeerName: "BrewUp IPA",
		Quantity: customtypes.NewQuantity(100, "Lt"),
	})

	var invalid muflone.ErrInvalid
	assert.ErrorAs(t, err, &invalid)
}

func TestFacadeSurfacesBusFailures(t *testing.T) {
	bus := muflone.NewServiceBus(slog.Default())
	require.NoError(t, bus.Close())
	facade := warehouses.NewFacade(bus, services.NewAvailabilityService())

	_, err := facade.UpdateAvailabilityDueToProductionOrder(context.Background(), warehouses.ProductionOrderJson{
		BeerId:   uuid.NewString(),
		BeerName: "BrewUp IPA",
		Quantity: customtypes.NewQuantity(100, "Lt"),
	})

	assert.Error(t, err)
}
