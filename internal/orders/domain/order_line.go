package domain

import (
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// BeerRef is an opaque reference to a catalog beer.
type BeerRef struct {
	shared.EntityID
}

func ParseBeerRef(raw string) (BeerRef, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return BeerRef{}, err
	}
	return BeerRef{EntityID: id}, nil
}

// OrderLine is an immutable value object: a quantity of one beer at the
// unit price in force when the order was placed.
type OrderLine struct {
	beerID    BeerRef
	units     int
	unitPrice shared.Money
}

func NewOrderLine(beerID BeerRef, units int, unitPrice shared.Money) (OrderLine, error) {
	if units <= 0 {
		return OrderLine{}, shared.NewValidationError("order line units must be positive")
	}
	if unitPrice.IsZero() {
		return OrderLine{}, shared.NewValidationError("order line needs a unit price")
	}
	return OrderLine{beerID: beerID, units: units, unitPrice: unitPrice}, nil
}

func (l OrderLine) BeerID() BeerRef         { return l.beerID }
func (l OrderLine) Units() int              { return l.units }
func (l OrderLine) UnitPrice() shared.Money { return l.unitPrice }

func (l OrderLine) Subtotal() shared.Money {
	return l.unitPrice.Mul(int64(l.units))
}
