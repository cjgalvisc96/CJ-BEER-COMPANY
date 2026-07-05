// Package domain models the catalog bounded context: the beers CJ Beer
// Company produces and sells.
package domain

import (
	"strings"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// BeerID is the catalog-scoped identity of a beer.
type BeerID struct {
	shared.EntityID
}

func NewBeerID() BeerID {
	return BeerID{EntityID: shared.NewEntityID()}
}

func ParseBeerID(raw string) (BeerID, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return BeerID{}, err
	}
	return BeerID{EntityID: id}, nil
}

type BeerStatus string

const (
	BeerStatusActive  BeerStatus = "active"
	BeerStatusRetired BeerStatus = "retired"
)

// Beer is the aggregate root of the catalog context. All invariants about a
// sellable beer (valid style, drinkable ABV, non-negative price) live here.
type Beer struct {
	shared.AggregateRoot

	id          BeerID
	name        string
	style       Style
	abv         ABV
	price       shared.Money
	description string
	status      BeerStatus
}

// NewBeer creates an active beer and records BeerCreated.
func NewBeer(name string, style Style, abv ABV, price shared.Money, description string) (*Beer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyBeerName
	}
	beer := &Beer{
		id:          NewBeerID(),
		name:        name,
		style:       style,
		abv:         abv,
		price:       price,
		description: strings.TrimSpace(description),
		status:      BeerStatusActive,
	}
	beer.RecordEvent(BeerCreated{
		BaseEvent: shared.NewBaseEvent(),
		BeerID:    beer.id.String(),
		Name:      beer.name,
		Style:     beer.style.String(),
	})
	return beer, nil
}

// Rehydrate rebuilds a Beer from persisted state without recording events.
func Rehydrate(
	id BeerID,
	name string,
	style Style,
	abv ABV,
	price shared.Money,
	description string,
	status BeerStatus,
) *Beer {
	return &Beer{
		id:          id,
		name:        name,
		style:       style,
		abv:         abv,
		price:       price,
		description: description,
		status:      status,
	}
}

func (b *Beer) ID() BeerID          { return b.id }
func (b *Beer) Name() string        { return b.name }
func (b *Beer) Style() Style        { return b.style }
func (b *Beer) ABV() ABV            { return b.abv }
func (b *Beer) Price() shared.Money { return b.price }
func (b *Beer) Description() string { return b.description }
func (b *Beer) Status() BeerStatus  { return b.status }
func (b *Beer) IsActive() bool      { return b.status == BeerStatusActive }

// ChangePrice updates the price and records BeerPriceChanged. Changing the
// price of a retired beer is a domain rule violation.
func (b *Beer) ChangePrice(newPrice shared.Money) error {
	if !b.IsActive() {
		return ErrBeerRetired
	}
	if b.price == newPrice {
		return nil
	}
	oldPrice := b.price
	b.price = newPrice
	b.RecordEvent(BeerPriceChanged{
		BaseEvent:     shared.NewBaseEvent(),
		BeerID:        b.id.String(),
		OldPriceCents: oldPrice.Cents(),
		NewPriceCents: newPrice.Cents(),
		Currency:      newPrice.Currency(),
	})
	return nil
}

// Retire removes the beer from sale. Retiring twice is idempotent.
func (b *Beer) Retire() {
	if !b.IsActive() {
		return
	}
	b.status = BeerStatusRetired
	b.RecordEvent(BeerRetired{
		BaseEvent: shared.NewBaseEvent(),
		BeerID:    b.id.String(),
		Name:      b.name,
	})
}
