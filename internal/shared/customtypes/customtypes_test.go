package customtypes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

func TestQuantity(t *testing.T) {
	quantity := customtypes.NewQuantity(100, "Lt")

	assert.Equal(t, 100, quantity.Value)
	assert.Equal(t, "Lt", quantity.UnitOfMeasure)
	assert.Equal(t, "100 Lt", quantity.String())
}

func TestPrice(t *testing.T) {
	price := customtypes.NewPrice(5, "EUR")

	assert.Equal(t, 5.0, price.Value)
	assert.Equal(t, "EUR", price.Currency)
	assert.Equal(t, "5.00 EUR", price.String())
}

func TestNewPageClampsBounds(t *testing.T) {
	assert.Equal(t, customtypes.Page{Limit: 50, Offset: 0}, customtypes.NewPage(0, 0), "defaults")
	assert.Equal(t, customtypes.Page{Limit: 200, Offset: 0}, customtypes.NewPage(9999, -5), "clamped")
	assert.Equal(t, customtypes.Page{Limit: 5, Offset: 10}, customtypes.NewPage(5, 10), "as requested")
}
