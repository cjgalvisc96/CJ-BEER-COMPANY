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
