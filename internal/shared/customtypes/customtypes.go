// Package customtypes holds the value objects shared by every module —
// the equivalent of BrewUp.Shared.CustomTypes.
package customtypes

import "fmt"

// Quantity is an amount in a unit of measure, e.g. Quantity{100, "Lt"}.
type Quantity struct {
	Value         int    `json:"value"`
	UnitOfMeasure string `json:"unit_of_measure"`
}

func NewQuantity(value int, unitOfMeasure string) Quantity {
	return Quantity{Value: value, UnitOfMeasure: unitOfMeasure}
}

func (q Quantity) String() string {
	return fmt.Sprintf("%d %s", q.Value, q.UnitOfMeasure)
}

// Pagination bounds: a page always has a sane, capped size.
const (
	DefaultPageLimit = 50
	MaxPageLimit     = 200
)

// Page is a read-model window. NewPage clamps out-of-range values instead
// of erroring — an over-eager limit is a preference, not a violation.
type Page struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func NewPage(limit, offset int) Page {
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}

// Price is an amount in a currency, e.g. Price{5, "EUR"}.
type Price struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

func NewPrice(value float64, currency string) Price {
	return Price{Value: value, Currency: currency}
}

func (p Price) String() string {
	return fmt.Sprintf("%.2f %s", p.Value, p.Currency)
}
