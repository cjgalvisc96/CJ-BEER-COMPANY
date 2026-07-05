package domain

import (
	"fmt"
	"regexp"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Money is an immutable value object storing an amount in the currency's
// minor unit (cents) to avoid floating-point drift.
type Money struct {
	cents    int64
	currency string
}

func NewMoney(cents int64, currency string) (Money, error) {
	if cents < 0 {
		return Money{}, NewValidationError("money amount cannot be negative")
	}
	if !currencyPattern.MatchString(currency) {
		return Money{}, NewValidationError("currency must be a 3-letter ISO code: " + currency)
	}
	return Money{cents: cents, currency: currency}, nil
}

func (m Money) Cents() int64 {
	return m.cents
}

func (m Money) Currency() string {
	return m.currency
}

func (m Money) Mul(factor int64) Money {
	return Money{cents: m.cents * factor, currency: m.currency}
}

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, NewValidationError(
			fmt.Sprintf("cannot add %s to %s", other.currency, m.currency),
		)
	}
	return Money{cents: m.cents + other.cents, currency: m.currency}, nil
}

func (m Money) IsZero() bool {
	return m == Money{}
}
