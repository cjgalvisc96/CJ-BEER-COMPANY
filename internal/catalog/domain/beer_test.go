package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

func mustMoney(t *testing.T, cents int64) shared.Money {
	t.Helper()
	money, err := shared.NewMoney(cents, "USD")
	require.NoError(t, err)
	return money
}

func newBeer(t *testing.T) *domain.Beer {
	t.Helper()
	abv, err := domain.NewABV(6.5)
	require.NoError(t, err)
	beer, err := domain.NewBeer("CJ Hop Bomb", domain.StyleIPA, abv, mustMoney(t, 550), "flagship IPA")
	require.NoError(t, err)
	return beer
}

func TestNewBeerRecordsCreatedEvent(t *testing.T) {
	beer := newBeer(t)

	events := beer.PullEvents()
	require.Len(t, events, 1)
	created, ok := events[0].(domain.BeerCreated)
	require.True(t, ok)
	assert.Equal(t, beer.ID().String(), created.BeerID)
	assert.Equal(t, "catalog.beer_created", created.EventName())
	assert.Empty(t, beer.PullEvents(), "events must be cleared after pull")
}

func TestNewBeerRejectsEmptyName(t *testing.T) {
	abv, err := domain.NewABV(5)
	require.NoError(t, err)

	_, err = domain.NewBeer("   ", domain.StyleLager, abv, mustMoney(t, 100), "")

	assert.ErrorIs(t, err, domain.ErrEmptyBeerName)
}

func TestChangePriceRecordsEventAndIsIdempotent(t *testing.T) {
	beer := newBeer(t)
	beer.PullEvents()

	require.NoError(t, beer.ChangePrice(mustMoney(t, 600)))
	require.NoError(t, beer.ChangePrice(mustMoney(t, 600)), "same price is a no-op")

	events := beer.PullEvents()
	require.Len(t, events, 1)
	changed := events[0].(domain.BeerPriceChanged)
	assert.Equal(t, int64(550), changed.OldPriceCents)
	assert.Equal(t, int64(600), changed.NewPriceCents)
}

func TestChangePriceOnRetiredBeerFails(t *testing.T) {
	beer := newBeer(t)
	beer.Retire()

	err := beer.ChangePrice(mustMoney(t, 700))

	assert.ErrorIs(t, err, domain.ErrBeerRetired)
}

func TestRetireIsIdempotent(t *testing.T) {
	beer := newBeer(t)
	beer.PullEvents()

	beer.Retire()
	beer.Retire()

	assert.Len(t, beer.PullEvents(), 1)
	assert.Equal(t, domain.BeerStatusRetired, beer.Status())
}

func TestParseStyle(t *testing.T) {
	style, err := domain.ParseStyle("  IPA ")
	require.NoError(t, err)
	assert.Equal(t, domain.StyleIPA, style)

	_, err = domain.ParseStyle("motor-oil")
	assert.Error(t, err)
}

func TestABVBounds(t *testing.T) {
	_, err := domain.NewABV(-1)
	assert.Error(t, err)

	_, err = domain.NewABV(21)
	assert.Error(t, err)

	abv, err := domain.NewABV(0)
	require.NoError(t, err)
	assert.Zero(t, abv.Value())
}
