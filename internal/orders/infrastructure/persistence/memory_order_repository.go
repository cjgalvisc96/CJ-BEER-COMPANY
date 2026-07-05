// Package persistence provides the in-memory adapter for the orders
// repository port.
package persistence

import (
	"context"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/domain"
)

type orderRecord struct {
	id           domain.OrderID
	customerName string
	lines        []domain.OrderLine
	status       domain.OrderStatus
	rejectReason string
	sequence     int
}

type MemoryOrderRepository struct {
	mu      sync.RWMutex
	orders  map[string]orderRecord
	counter int
}

func NewMemoryOrderRepository() *MemoryOrderRepository {
	return &MemoryOrderRepository{orders: make(map[string]orderRecord)}
}

func (r *MemoryOrderRepository) Save(_ context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := order.ID().String()
	sequence := r.counter
	if existing, ok := r.orders[key]; ok {
		sequence = existing.sequence
	} else {
		r.counter++
	}
	r.orders[key] = orderRecord{
		id:           order.ID(),
		customerName: order.CustomerName(),
		lines:        order.Lines(),
		status:       order.Status(),
		rejectReason: order.RejectReason(),
		sequence:     sequence,
	}
	return nil
}

func (r *MemoryOrderRepository) FindByID(_ context.Context, id domain.OrderID) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.orders[id.String()]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return record.toEntity(), nil
}

func (r *MemoryOrderRepository) FindAll(_ context.Context) ([]*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	records := make([]orderRecord, 0, len(r.orders))
	for _, record := range r.orders {
		records = append(records, record)
	}
	// Stable insertion order beats map randomness for API consumers.
	for i := 1; i < len(records); i++ {
		for j := i; j > 0 && records[j].sequence < records[j-1].sequence; j-- {
			records[j], records[j-1] = records[j-1], records[j]
		}
	}
	orders := make([]*domain.Order, 0, len(records))
	for _, record := range records {
		orders = append(orders, record.toEntity())
	}
	return orders, nil
}

func (record orderRecord) toEntity() *domain.Order {
	lines := make([]domain.OrderLine, len(record.lines))
	copy(lines, record.lines)
	return domain.RehydrateOrder(record.id, record.customerName, lines, record.status, record.rejectReason)
}
