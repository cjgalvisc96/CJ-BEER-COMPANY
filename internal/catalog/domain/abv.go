package domain

import (
	"fmt"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// maxABV is the highest alcohol-by-volume the brewery can legally sell.
const maxABV = 20.0

// ABV is alcohol by volume as a percentage, e.g. 5.4.
type ABV struct {
	value float64
}

func NewABV(value float64) (ABV, error) {
	if value < 0 || value > maxABV {
		return ABV{}, shared.NewValidationError(
			fmt.Sprintf("abv must be between 0 and %.0f, got %.2f", maxABV, value),
		)
	}
	return ABV{value: value}, nil
}

func (a ABV) Value() float64 {
	return a.value
}
