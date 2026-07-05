package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/domain"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

func newStockItem(t *testing.T, reorderLevel int) *domain.StockItem {
	t.Helper()
	beerID, err := domain.ParseBeerRef(shared.NewEntityID().String())
	require.NoError(t, err)
	item, err := domain.NewStockItem(beerID, reorderLevel)
	require.NoError(t, err)
	return item
}

func TestReplenishIncreasesQuantity(t *testing.T) {
	item := newStockItem(t, 0)

	require.NoError(t, item.Replenish(100))

	assert.Equal(t, 100, item.Quantity())
	events := item.PullEvents()
	require.Len(t, events, 1)
	assert.Equal(t, "inventory.stock_replenished", events[0].EventName())
}

func TestReserveDecreasesQuantity(t *testing.T) {
	item := newStockItem(t, 0)
	require.NoError(t, item.Replenish(50))
	item.PullEvents()

	require.NoError(t, item.Reserve(20))

	assert.Equal(t, 30, item.Quantity())
}

func TestReserveMoreThanAvailableFails(t *testing.T) {
	item := newStockItem(t, 0)
	require.NoError(t, item.Replenish(10))

	err := item.Reserve(11)

	assert.ErrorIs(t, err, domain.ErrInsufficientStock)
	assert.Equal(t, 10, item.Quantity(), "failed reserve must not mutate stock")
}

func TestReserveCrossingReorderLevelWarns(t *testing.T) {
	item := newStockItem(t, 5)
	require.NoError(t, item.Replenish(10))
	item.PullEvents()

	require.NoError(t, item.Reserve(6))

	events := item.PullEvents()
	require.Len(t, events, 2)
	assert.Equal(t, "inventory.stock_reserved", events[0].EventName())
	assert.Equal(t, "inventory.stock_level_low", events[1].EventName())
}

func TestNonPositiveAmountsAreRejected(t *testing.T) {
	item := newStockItem(t, 0)

	assert.Error(t, item.Replenish(0))
	assert.Error(t, item.Reserve(-3))
}
