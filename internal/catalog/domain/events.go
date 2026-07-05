package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

// Topic names are the public contract of this context; other contexts
// subscribe to these strings, never to these types.
const (
	BeerCreatedTopic      = "catalog.beer_created"
	BeerPriceChangedTopic = "catalog.beer_price_changed"
	BeerRetiredTopic      = "catalog.beer_retired"
)

type BeerCreated struct {
	shared.BaseEvent
	BeerID string `json:"beer_id"`
	Name   string `json:"name"`
	Style  string `json:"style"`
}

func (BeerCreated) EventName() string { return BeerCreatedTopic }

type BeerPriceChanged struct {
	shared.BaseEvent
	BeerID        string `json:"beer_id"`
	OldPriceCents int64  `json:"old_price_cents"`
	NewPriceCents int64  `json:"new_price_cents"`
	Currency      string `json:"currency"`
}

func (BeerPriceChanged) EventName() string { return BeerPriceChangedTopic }

type BeerRetired struct {
	shared.BaseEvent
	BeerID string `json:"beer_id"`
	Name   string `json:"name"`
}

func (BeerRetired) EventName() string { return BeerRetiredTopic }
