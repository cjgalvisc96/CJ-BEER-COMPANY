package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
)

func TestSalesOrderServiceQueries(t *testing.T) {
	service := services.NewSalesOrderService()
	ctx := context.Background()

	_, found := service.GetSalesOrder(ctx, "missing")
	assert.False(t, found)
	assert.Empty(t, service.GetSalesOrders(ctx))

	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "1", SalesOrderNumber: "A"}))
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "2", SalesOrderNumber: "B"}))
	require.NoError(t, service.CreateSalesOrder(ctx, dtos.SalesOrder{Id: "1", SalesOrderNumber: "A2"}),
		"upsert keeps the original position")

	order, found := service.GetSalesOrder(ctx, "1")
	require.True(t, found)
	assert.Equal(t, "A2", order.SalesOrderNumber)

	all := service.GetSalesOrders(ctx)
	require.Len(t, all, 2)
	assert.Equal(t, "A2", all[0].SalesOrderNumber, "insertion order is stable")
}
