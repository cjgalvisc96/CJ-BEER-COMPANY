// Package persistence provides the in-memory adapter for the brewing
// repository port.
package persistence

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
)

type batchRecord struct {
	id          domain.BatchID
	beerID      domain.BeerRef
	units       int
	status      domain.BatchStatus
	startedAt   time.Time
	completedAt *time.Time
}

type MemoryBatchRepository struct {
	mu      sync.RWMutex
	batches map[string]batchRecord
}

func NewMemoryBatchRepository() *MemoryBatchRepository {
	return &MemoryBatchRepository{batches: make(map[string]batchRecord)}
}

func (r *MemoryBatchRepository) Save(_ context.Context, batch *domain.Batch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var completedAt *time.Time
	if batch.CompletedAt() != nil {
		at := *batch.CompletedAt()
		completedAt = &at
	}
	r.batches[batch.ID().String()] = batchRecord{
		id:          batch.ID(),
		beerID:      batch.BeerID(),
		units:       batch.Units(),
		status:      batch.Status(),
		startedAt:   batch.StartedAt(),
		completedAt: completedAt,
	}
	return nil
}

func (r *MemoryBatchRepository) FindByID(_ context.Context, id domain.BatchID) (*domain.Batch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.batches[id.String()]
	if !ok {
		return nil, domain.ErrBatchNotFound
	}
	return record.toEntity(), nil
}

func (r *MemoryBatchRepository) FindAll(_ context.Context) ([]*domain.Batch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	batches := make([]*domain.Batch, 0, len(r.batches))
	for _, record := range r.batches {
		batches = append(batches, record.toEntity())
	}
	sort.Slice(batches, func(i, j int) bool {
		return batches[i].StartedAt().Before(batches[j].StartedAt())
	})
	return batches, nil
}

func (record batchRecord) toEntity() *domain.Batch {
	var completedAt *time.Time
	if record.completedAt != nil {
		at := *record.completedAt
		completedAt = &at
	}
	return domain.RehydrateBatch(
		record.id, record.beerID, record.units, record.status, record.startedAt, completedAt,
	)
}
