// Package integrationevents holds the events the Sales module shares with
// other bounded contexts. Deliberately separate types from the domain
// events, even when the shape is similar: the book warns that reusing a
// domain event as an integration event couples the module's ubiquitous
// language to the outside world (Chapter 4, "Domain and integration
// events").
package integrationevents

import (
	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
)

const SalesOrderCreatedName = "sales.sales_order_created"

// SalesOrderCreated carries only the essential data other contexts need
// to react (the warehouse allocates stock per row).
type SalesOrderCreated struct {
	muflone.IntegrationEventBase
	SalesOrderId string                 `json:"sales_order_id"`
	Rows         []SalesOrderCreatedRow `json:"rows"`
}

type SalesOrderCreatedRow struct {
	BeerId        string `json:"beer_id"`
	BeerName      string `json:"beer_name"`
	Quantity      int    `json:"quantity"`
	UnitOfMeasure string `json:"unit_of_measure"`
}

func NewSalesOrderCreated(salesOrderId uuid.UUID, commitId uuid.UUID, rows []SalesOrderCreatedRow) SalesOrderCreated {
	return SalesOrderCreated{
		IntegrationEventBase: muflone.NewIntegrationEventBase(commitId),
		SalesOrderId:         salesOrderId.String(),
		Rows:                 rows,
	}
}

func (SalesOrderCreated) MessageName() string { return SalesOrderCreatedName }
