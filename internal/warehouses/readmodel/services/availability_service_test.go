package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func TestAvailabilityServiceQueries(t *testing.T) {
	service := services.NewAvailabilityService()
	ctx := context.Background()

	_, found := service.GetAvailability(ctx, "missing")
	assert.False(t, found)
	assert.Empty(t, service.GetAvailabilities(ctx))

	require.NoError(t, service.UpsertAvailability(ctx, dtos.Availability{
		BeerId: "b", BeerName: "Zeta Stout", Quantity: customtypes.NewQuantity(10, "Lt"),
	}))
	require.NoError(t, service.UpsertAvailability(ctx, dtos.Availability{
		BeerId: "a", BeerName: "Alpha IPA", Quantity: customtypes.NewQuantity(20, "Lt"),
	}))

	availability, found := service.GetAvailability(ctx, "a")
	require.True(t, found)
	assert.Equal(t, 20, availability.Quantity.Value)

	all := service.GetAvailabilities(ctx)
	require.Len(t, all, 2)
	assert.Equal(t, "Alpha IPA", all[0].BeerName, "listing is sorted by beer name")
}
