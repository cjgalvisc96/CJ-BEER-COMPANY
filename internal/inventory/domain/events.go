package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

const (
	StockReplenishedTopic = "inventory.stock_replenished"
	StockReservedTopic    = "inventory.stock_reserved"
	StockLevelLowTopic    = "inventory.stock_level_low"
)

type StockReplenished struct {
	shared.BaseEvent
	BeerID   string `json:"beer_id"`
	Units    int    `json:"units"`
	Quantity int    `json:"quantity"`
}

func (StockReplenished) EventName() string { return StockReplenishedTopic }

type StockReserved struct {
	shared.BaseEvent
	BeerID   string `json:"beer_id"`
	Units    int    `json:"units"`
	Quantity int    `json:"quantity"`
}

func (StockReserved) EventName() string { return StockReservedTopic }

type StockLevelLow struct {
	shared.BaseEvent
	BeerID       string `json:"beer_id"`
	Quantity     int    `json:"quantity"`
	ReorderLevel int    `json:"reorder_level"`
}

func (StockLevelLow) EventName() string { return StockLevelLowTopic }
