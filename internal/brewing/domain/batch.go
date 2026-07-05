// Package domain models the brewing bounded context: production batches
// that turn recipes into sellable units.
package domain

import (
	"time"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type BatchID struct {
	shared.EntityID
}

func NewBatchID() BatchID {
	return BatchID{EntityID: shared.NewEntityID()}
}

func ParseBatchID(raw string) (BatchID, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return BatchID{}, err
	}
	return BatchID{EntityID: id}, nil
}

// BeerRef is an opaque reference to a catalog beer.
type BeerRef struct {
	shared.EntityID
}

func ParseBeerRef(raw string) (BeerRef, error) {
	id, err := shared.ParseEntityID(raw)
	if err != nil {
		return BeerRef{}, err
	}
	return BeerRef{EntityID: id}, nil
}

type BatchStatus string

const (
	BatchStatusBrewing   BatchStatus = "brewing"
	BatchStatusCompleted BatchStatus = "completed"
)

// Batch is the aggregate root: one production run of one beer.
type Batch struct {
	shared.AggregateRoot

	id          BatchID
	beerID      BeerRef
	units       int
	status      BatchStatus
	startedAt   time.Time
	completedAt *time.Time
}

// StartBatch begins brewing a number of units of a beer.
func StartBatch(beerID BeerRef, units int) (*Batch, error) {
	if units <= 0 {
		return nil, shared.NewValidationError("batch units must be positive")
	}
	batch := &Batch{
		id:        NewBatchID(),
		beerID:    beerID,
		units:     units,
		status:    BatchStatusBrewing,
		startedAt: time.Now().UTC(),
	}
	batch.RecordEvent(BatchStarted{
		BaseEvent: shared.NewBaseEvent(),
		BatchID:   batch.id.String(),
		BeerID:    beerID.String(),
		Units:     units,
	})
	return batch, nil
}

// RehydrateBatch rebuilds a Batch from persisted state.
func RehydrateBatch(
	id BatchID,
	beerID BeerRef,
	units int,
	status BatchStatus,
	startedAt time.Time,
	completedAt *time.Time,
) *Batch {
	return &Batch{
		id:          id,
		beerID:      beerID,
		units:       units,
		status:      status,
		startedAt:   startedAt,
		completedAt: completedAt,
	}
}

func (b *Batch) ID() BatchID             { return b.id }
func (b *Batch) BeerID() BeerRef         { return b.beerID }
func (b *Batch) Units() int              { return b.units }
func (b *Batch) Status() BatchStatus     { return b.status }
func (b *Batch) StartedAt() time.Time    { return b.startedAt }
func (b *Batch) CompletedAt() *time.Time { return b.completedAt }

// Complete finishes the batch with the units that actually came out of the
// tanks (spillage and quality control mean it can differ from the plan) and
// records BatchCompleted — the event inventory listens to.
func (b *Batch) Complete(producedUnits int) error {
	if b.status == BatchStatusCompleted {
		return ErrBatchAlreadyCompleted
	}
	if producedUnits <= 0 {
		return shared.NewValidationError("produced units must be positive")
	}
	now := time.Now().UTC()
	b.status = BatchStatusCompleted
	b.units = producedUnits
	b.completedAt = &now
	b.RecordEvent(BatchCompleted{
		BaseEvent: shared.NewBaseEvent(),
		BatchID:   b.id.String(),
		BeerID:    b.beerID.String(),
		Units:     producedUnits,
	})
	return nil
}
