// Package services holds the Sales read-model services —
// BrewUp.Sales.ReadModel/Services: the projection writer used by event
// handlers and the query service used by the facade.
package services

import (
	"context"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
)

// SalesOrderService is both ISalesOrderService (projection updates) and
// SalesOrderQueries (reads) over an in-memory store; swapping to a real
// query database only touches this file.
type SalesOrderService struct {
	mu     sync.RWMutex
	orders map[string]dtos.SalesOrder
	order  []string
}

func NewSalesOrderService() *SalesOrderService {
	return &SalesOrderService{orders: make(map[string]dtos.SalesOrder)}
}

// CreateSalesOrder upserts the projection for a newly created order.
func (s *SalesOrderService) CreateSalesOrder(_ context.Context, order dtos.SalesOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.orders[order.Id]; !exists {
		s.order = append(s.order, order.Id)
	}
	s.orders[order.Id] = order
	return nil
}

func (s *SalesOrderService) GetSalesOrder(_ context.Context, id string) (dtos.SalesOrder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	return order, ok
}

func (s *SalesOrderService) GetSalesOrders(_ context.Context) []dtos.SalesOrder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]dtos.SalesOrder, 0, len(s.order))
	for _, id := range s.order {
		orders = append(orders, s.orders[id])
	}
	return orders
}
