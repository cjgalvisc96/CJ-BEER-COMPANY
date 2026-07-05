package commands_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/infrastructure/persistence"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// fakeCatalog and spyPublisher are hand-rolled test doubles: the ports are
// small enough that a mocking library would be more code than this.
type fakeCatalog struct {
	beers map[string]ports.BeerSnapshot
}

func (f *fakeCatalog) FindBeer(_ context.Context, beerID string) (ports.BeerSnapshot, error) {
	snapshot, ok := f.beers[beerID]
	if !ok {
		return ports.BeerSnapshot{}, shared.NewNotFoundError("beer not found")
	}
	return snapshot, nil
}

type spyPublisher struct {
	published []shared.Event
}

func (s *spyPublisher) Publish(_ context.Context, events ...shared.Event) error {
	s.published = append(s.published, events...)
	return nil
}

func TestPlaceOrderCapturesCatalogPrice(t *testing.T) {
	beerID := shared.NewEntityID().String()
	price, err := shared.NewMoney(725, "USD")
	require.NoError(t, err)
	catalog := &fakeCatalog{beers: map[string]ports.BeerSnapshot{
		beerID: {ID: beerID, Name: "CJ IPA", Price: price, Sellable: true},
	}}
	publisher := &spyPublisher{}
	handler := commands.NewPlaceOrderHandler(persistence.NewMemoryOrderRepository(), catalog, publisher)

	order, err := handler.Handle(context.Background(), commands.PlaceOrderInput{
		CustomerName: "Cristian",
		Lines:        []commands.PlaceOrderLine{{BeerID: beerID, Units: 4}},
	})

	require.NoError(t, err)
	assert.Equal(t, int64(4*725), order.TotalCents)
	assert.Equal(t, "pending", order.Status)
	require.Len(t, publisher.published, 1)
	assert.Equal(t, "orders.order_placed", publisher.published[0].EventName())
}

func TestPlaceOrderRejectsUnknownAndRetiredBeers(t *testing.T) {
	knownID := shared.NewEntityID().String()
	price, err := shared.NewMoney(500, "USD")
	require.NoError(t, err)
	catalog := &fakeCatalog{beers: map[string]ports.BeerSnapshot{
		knownID: {ID: knownID, Name: "Retired Ale", Price: price, Sellable: false},
	}}
	handler := commands.NewPlaceOrderHandler(
		persistence.NewMemoryOrderRepository(), catalog, &spyPublisher{},
	)

	_, err = handler.Handle(context.Background(), commands.PlaceOrderInput{
		CustomerName: "Cristian",
		Lines:        []commands.PlaceOrderLine{{BeerID: shared.NewEntityID().String(), Units: 1}},
	})
	assert.ErrorIs(t, err, domain.ErrBeerNotSellable)

	_, err = handler.Handle(context.Background(), commands.PlaceOrderInput{
		CustomerName: "Cristian",
		Lines:        []commands.PlaceOrderLine{{BeerID: knownID, Units: 1}},
	})
	assert.ErrorIs(t, err, domain.ErrBeerNotSellable)
}
