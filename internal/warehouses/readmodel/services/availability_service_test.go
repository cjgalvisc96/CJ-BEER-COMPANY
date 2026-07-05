package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func TestAvailabilityServiceQueries(t *testing.T) {
	service := services.NewAvailabilityService()
	ctx := context.Background()

	_, err := service.GetAvailability(ctx, "missing")
	assert.ErrorIs(t, err, muflone.ErrNotFound)
	empty, err := service.GetAvailabilities(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)

	require.NoError(t, service.UpsertAvailability(ctx, dtos.Availability{
		BeerId: "b", BeerName: "Zeta Stout", Quantity: customtypes.NewQuantity(10, "Lt"),
	}))
	require.NoError(t, service.UpsertAvailability(ctx, dtos.Availability{
		BeerId: "a", BeerName: "Alpha IPA", Quantity: customtypes.NewQuantity(20, "Lt"),
	}))

	availability, err := service.GetAvailability(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, 20, availability.Quantity.Value)

	all, err := service.GetAvailabilities(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "Alpha IPA", all[0].BeerName, "listing is sorted by beer name")
}
