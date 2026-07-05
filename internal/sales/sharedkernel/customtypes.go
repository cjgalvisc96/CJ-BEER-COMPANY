// Package sharedkernel is the published language of the Sales module —
// the equivalent of BrewUp.Sales.SharedKernel: the custom types, commands,
// and events other layers of the module (and only this module) build on.
package sharedkernel

import (
	"time"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

// The custom types mirror the book's records: strongly named wrappers with
// a Value, so a SalesOrderId can never be confused with a CustomerId.

type SalesOrderId struct {
	Value uuid.UUID `json:"value"`
}

func NewSalesOrderId() SalesOrderId {
	return SalesOrderId{Value: uuid.New()}
}

type SalesOrderNumber struct {
	Value string `json:"value"`
}

type OrderDate struct {
	Value time.Time `json:"value"`
}

type CustomerId struct {
	Value uuid.UUID `json:"value"`
}

type CustomerName struct {
	Value string `json:"value"`
}

type BeerId struct {
	Value uuid.UUID `json:"value"`
}

type BeerName struct {
	Value string `json:"value"`
}

// SalesOrderRowDto is one line of a sales order as it travels in commands
// and events.
type SalesOrderRowDto struct {
	BeerId   BeerId               `json:"beer_id"`
	BeerName BeerName             `json:"beer_name"`
	Quantity customtypes.Quantity `json:"quantity"`
	Price    customtypes.Price    `json:"price"`
}
