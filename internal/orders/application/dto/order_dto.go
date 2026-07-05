// Package dto defines the data shapes crossing the orders application
// boundary.
package dto

import "github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"

type OrderLineOutput struct {
	BeerID         string `json:"beer_id"`
	Units          int    `json:"units"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	SubtotalCents  int64  `json:"subtotal_cents"`
}

type OrderOutput struct {
	ID           string            `json:"id"`
	CustomerName string            `json:"customer_name"`
	Lines        []OrderLineOutput `json:"lines"`
	Status       string            `json:"status"`
	RejectReason string            `json:"reject_reason,omitempty"`
	TotalCents   int64             `json:"total_cents"`
	Currency     string            `json:"currency"`
}

func OrderOutputFromEntity(order *domain.Order) (OrderOutput, error) {
	total, err := order.Total()
	if err != nil {
		return OrderOutput{}, err
	}
	lines := make([]OrderLineOutput, 0, len(order.Lines()))
	for _, line := range order.Lines() {
		lines = append(lines, OrderLineOutput{
			BeerID:         line.BeerID().String(),
			Units:          line.Units(),
			UnitPriceCents: line.UnitPrice().Cents(),
			SubtotalCents:  line.Subtotal().Cents(),
		})
	}
	return OrderOutput{
		ID:           order.ID().String(),
		CustomerName: order.CustomerName(),
		Lines:        lines,
		Status:       string(order.Status()),
		RejectReason: order.RejectReason(),
		TotalCents:   total.Cents(),
		Currency:     total.Currency(),
	}, nil
}
