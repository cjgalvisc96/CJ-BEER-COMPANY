package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

func TestSalesOrderServiceQueries(t *testing.T) {
	service := services.NewSalesOrderService()
	ctx := context.Background()

	_, err := service.GetSalesOrder(ctx, "missing")
	assert.ErrorIs(t, err, muflone.ErrNotFound)
	empty, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
	require.NoError(t, err)
	assert.Empty(t, empty)

	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "1", SalesOrderNumber: "A"}))
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "2", SalesOrderNumber: "B"}))
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "1", SalesOrderNumber: "A2"}),
		"upsert keeps the original position")

	order, err := service.GetSalesOrder(ctx, "1")
	require.NoError(t, err)
	assert.Equal(t, "A2", order.SalesOrderNumber)

	all, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "A2", all[0].SalesOrderNumber, "insertion order is stable")
}

func TestAllocationStatusProjection(t *testing.T) {
	service := services.NewSalesOrderService()
	ctx := context.Background()

	// The outcome can land before the created projection: a stub absorbs it.
	require.NoError(t, service.UpdateAllocationStatus(ctx, "early", "rejected", "shortage"))
	order, err := service.GetSalesOrder(ctx, "early")
	require.NoError(t, err)
	assert.Equal(t, "rejected", order.AllocationStatus)

	// The created projection then merges in WITHOUT losing the settled status.
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "early", SalesOrderNumber: "X"}))
	order, err = service.GetSalesOrder(ctx, "early")
	require.NoError(t, err)
	assert.Equal(t, "X", order.SalesOrderNumber)
	assert.Equal(t, "rejected", order.AllocationStatus)
	assert.Equal(t, "shortage", order.RejectionReason)

	// The usual direction: created first (pending), settled later.
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "usual"}))
	order, err = service.GetSalesOrder(ctx, "usual")
	require.NoError(t, err)
	assert.Equal(t, "pending", order.AllocationStatus)
	require.NoError(t, service.UpdateAllocationStatus(ctx, "usual", "allocated", ""))
	order, err = service.GetSalesOrder(ctx, "usual")
	require.NoError(t, err)
	assert.Equal(t, "allocated", order.AllocationStatus)
}
