// Package dtos holds the Sales read-model DTOs — BrewUp.Sales.ReadModel/
// Dtos. They mirror the domain shapes but exist purely for querying,
// without domain behavior.
package dtos

import (
	"time"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

type SalesOrder struct {
	Id               string          `json:"id"`
	SalesOrderNumber string          `json:"sales_order_number"`
	OrderDate        time.Time       `json:"order_date"`
	CustomerId       string          `json:"customer_id"`
	CustomerName     string          `json:"customer_name"`
	Rows             []SalesOrderRow `json:"rows"`
}

type SalesOrderRow struct {
	BeerId   string               `json:"beer_id"`
	BeerName string               `json:"beer_name"`
	Quantity customtypes.Quantity `json:"quantity"`
	Price    customtypes.Price    `json:"price"`
}
