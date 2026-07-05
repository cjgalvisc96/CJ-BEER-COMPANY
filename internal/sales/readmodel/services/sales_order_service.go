// Package services holds the Sales read-model services —
// BrewUp.Sales.ReadModel/Services: the projection writer used by event
// handlers and the query service used by the facade. Two adapters exist:
// in-memory (dev/tests) and Postgres (production); both implement the
// same behavior.
package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

// SalesOrderService is the in-memory adapter.
type SalesOrderService struct {
	mu     sync.RWMutex
	orders map[string]dtos.SalesOrder
	order  []string
}

func NewSalesOrderService() *SalesOrderService {
	return &SalesOrderService{orders: make(map[string]dtos.SalesOrder)}
}

// CreateSalesOrder upserts the projection for a newly created order. A
// status that arrived first (event ordering across streams is not
// guaranteed) is preserved.
func (s *SalesOrderService) CreateSalesOrder(_ context.Context, order dtos.SalesOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.orders[order.Id]
	if !exists {
		s.order = append(s.order, order.Id)
	}
	if order.AllocationStatus == "" {
		order.AllocationStatus = "pending"
	}
	if exists && existing.AllocationStatus != "" && existing.AllocationStatus != "pending" {
		order.AllocationStatus = existing.AllocationStatus
		order.RejectionReason = existing.RejectionReason
	}
	s.orders[order.Id] = order
	return nil
}

// UpdateAllocationStatus records the saga outcome; if the created
// projection has not landed yet, a stub row holds the status until it does.
func (s *SalesOrderService) UpdateAllocationStatus(_ context.Context, salesOrderId, status, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, exists := s.orders[salesOrderId]
	if !exists {
		s.order = append(s.order, salesOrderId)
		order = dtos.SalesOrder{Id: salesOrderId}
	}
	order.AllocationStatus = status
	order.RejectionReason = reason
	s.orders[salesOrderId] = order
	return nil
}

func (s *SalesOrderService) GetSalesOrder(_ context.Context, id string) (dtos.SalesOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	if !ok {
		return dtos.SalesOrder{}, fmt.Errorf("%w: sales order %s", muflone.ErrNotFound, id)
	}
	return order, nil
}

func (s *SalesOrderService) GetSalesOrders(_ context.Context, page customtypes.Page) ([]dtos.SalesOrder, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	orders := make([]dtos.SalesOrder, 0, page.Limit)
	for index := page.Offset; index < len(s.order) && len(orders) < page.Limit; index++ {
		orders = append(orders, s.orders[s.order[index]])
	}
	return orders, nil
}
