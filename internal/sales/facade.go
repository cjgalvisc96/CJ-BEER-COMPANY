// Package sales is the Sales bounded context of CJ Beer Company —
// the Go rendition of the book's BrewUp Sales module.
package sales

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/sharedkernel/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

// SalesOrderJson is the inbound payload for creating an order — the
// book's SalesOrderJson body posted to /v1/sales. Id is optional: clients
// that supply their own uuid get IDEMPOTENT creation (safe retries).
type SalesOrderJson struct {
	Id               string              `json:"id"`
	SalesOrderNumber string              `json:"sales_order_number" binding:"required"`
	OrderDate        time.Time           `json:"order_date"`
	CustomerId       string              `json:"customer_id"`
	CustomerName     string              `json:"customer_name" binding:"required"`
	Rows             []SalesOrderRowJson `json:"rows" binding:"required,dive"`
}

type SalesOrderRowJson struct {
	BeerId   string               `json:"beer_id" binding:"required"`
	BeerName string               `json:"beer_name" binding:"required"`
	Quantity customtypes.Quantity `json:"quantity"`
	Price    customtypes.Price    `json:"price"`
}

// SalesOrderQueries is the read-side port of the facade; the in-memory
// and Postgres read-model services both implement it.
type SalesOrderQueries interface {
	GetSalesOrder(ctx context.Context, id string) (dtos.SalesOrder, error)
	GetSalesOrders(ctx context.Context, page customtypes.Page) ([]dtos.SalesOrder, error)
}

// Facade is the module's public surface (the book's ISalesFacade): the
// REST layer talks only to facades, never to a module's internals.
type Facade struct {
	bus     *muflone.ServiceBus
	queries SalesOrderQueries
}

func NewFacade(bus *muflone.ServiceBus, queries SalesOrderQueries) *Facade {
	return &Facade{bus: bus, queries: queries}
}

// CreateSalesOrder validates the payload into a CreateSalesOrder command
// and sends it on the bus (fire-and-forget). It returns the pre-generated
// order id — the caller polls the read model for the projection.
func (f *Facade) CreateSalesOrder(ctx context.Context, body SalesOrderJson) (string, error) {
	salesOrderId := sharedkernel.NewSalesOrderId()
	if body.Id != "" {
		parsed, err := uuid.Parse(body.Id)
		if err != nil {
			return "", muflone.ErrInvalid("invalid sales order id: " + body.Id)
		}
		salesOrderId = sharedkernel.SalesOrderId{Value: parsed}
	}
	customerId, err := uuid.Parse(body.CustomerId)
	if err != nil {
		customerId = uuid.New()
	}
	orderDate := body.OrderDate
	if orderDate.IsZero() {
		orderDate = time.Now().UTC()
	}

	rows := make([]sharedkernel.SalesOrderRowDto, 0, len(body.Rows))
	for _, row := range body.Rows {
		beerId, err := uuid.Parse(row.BeerId)
		if err != nil {
			return "", muflone.ErrInvalid("invalid beer id: " + row.BeerId)
		}
		rows = append(rows, sharedkernel.SalesOrderRowDto{
			BeerId:   sharedkernel.BeerId{Value: beerId},
			BeerName: sharedkernel.BeerName{Value: row.BeerName},
			Quantity: row.Quantity,
			Price:    row.Price,
		})
	}

	command := commands.NewCreateSalesOrder(
		salesOrderId,
		uuid.New(),
		sharedkernel.SalesOrderNumber{Value: body.SalesOrderNumber},
		sharedkernel.OrderDate{Value: orderDate},
		sharedkernel.CustomerId{Value: customerId},
		sharedkernel.CustomerName{Value: body.CustomerName},
		rows,
	)
	if err := f.bus.Send(ctx, command); err != nil {
		return "", err
	}
	return salesOrderId.Value.String(), nil
}

func (f *Facade) GetSalesOrder(ctx context.Context, id string) (dtos.SalesOrder, error) {
	return f.queries.GetSalesOrder(ctx, id)
}

func (f *Facade) GetSalesOrders(ctx context.Context, page customtypes.Page) ([]dtos.SalesOrder, error) {
	return f.queries.GetSalesOrders(ctx, page)
}
