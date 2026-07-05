package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

func newLine(t *testing.T, units int, priceCents int64) domain.OrderLine {
	t.Helper()
	beerID, err := domain.ParseBeerRef(shared.NewEntityID().String())
	require.NoError(t, err)
	price, err := shared.NewMoney(priceCents, "USD")
	require.NoError(t, err)
	line, err := domain.NewOrderLine(beerID, units, price)
	require.NoError(t, err)
	return line
}

func placeOrder(t *testing.T) *domain.Order {
	t.Helper()
	order, err := domain.PlaceOrder("Cristian", []domain.OrderLine{
		newLine(t, 2, 500),
		newLine(t, 3, 400),
	})
	require.NoError(t, err)
	return order
}

func TestPlaceOrderComputesTotalAndRecordsEvent(t *testing.T) {
	order := placeOrder(t)

	total, err := order.Total()
	require.NoError(t, err)
	assert.Equal(t, int64(2*500+3*400), total.Cents())
	assert.Equal(t, domain.OrderStatusPending, order.Status())

	events := order.PullEvents()
	require.Len(t, events, 1)
	placed := events[0].(domain.OrderPlaced)
	assert.Equal(t, "orders.order_placed", placed.EventName())
	assert.Len(t, placed.Lines, 2)
	assert.Equal(t, total.Cents(), placed.TotalCents)
}

func TestPlaceOrderValidations(t *testing.T) {
	_, err := domain.PlaceOrder("", []domain.OrderLine{newLine(t, 1, 100)})
	assert.ErrorIs(t, err, domain.ErrEmptyCustomerName)

	_, err = domain.PlaceOrder("Cristian", nil)
	assert.ErrorIs(t, err, domain.ErrEmptyOrder)
}

func TestConfirmOnlyFromPending(t *testing.T) {
	order := placeOrder(t)

	require.NoError(t, order.Confirm())
	assert.Equal(t, domain.OrderStatusConfirmed, order.Status())

	assert.ErrorIs(t, order.Confirm(), domain.ErrOrderNotPending)
}

func TestRejectStoresReason(t *testing.T) {
	order := placeOrder(t)

	require.NoError(t, order.Reject("insufficient stock"))

	assert.Equal(t, domain.OrderStatusRejected, order.Status())
	assert.Equal(t, "insufficient stock", order.RejectReason())
	assert.ErrorIs(t, order.Cancel(), domain.ErrOrderNotCancellable)
}

func TestCancelFromPendingAndConfirmed(t *testing.T) {
	pending := placeOrder(t)
	require.NoError(t, pending.Cancel())
	assert.Equal(t, domain.OrderStatusCancelled, pending.Status())

	confirmed := placeOrder(t)
	require.NoError(t, confirmed.Confirm())
	require.NoError(t, confirmed.Cancel())
	assert.Equal(t, domain.OrderStatusCancelled, confirmed.Status())
}

func TestOrderLineValidations(t *testing.T) {
	beerID, err := domain.ParseBeerRef(shared.NewEntityID().String())
	require.NoError(t, err)
	price, err := shared.NewMoney(100, "USD")
	require.NoError(t, err)

	_, err = domain.NewOrderLine(beerID, 0, price)
	assert.Error(t, err)

	_, err = domain.NewOrderLine(beerID, 1, shared.Money{})
	assert.Error(t, err)
}
