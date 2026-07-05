package muflone

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Repository is Muflone's IRepository: only GetByID and Save. The write
// model is optimized for writes — queries belong to the read model.
type Repository[T Aggregate] interface {
	GetByID(ctx context.Context, id uuid.UUID) (T, error)
	Save(ctx context.Context, aggregate T, commitID uuid.UUID) error
}

// DomainEventPublisher lets the repository hand freshly persisted events to
// the service bus, where read-model projections and integration publishers
// subscribe. Nil-able for specification tests.
type DomainEventPublisher interface {
	PublishDomainEvent(ctx context.Context, event DomainEvent) error
}

// EventStoreRepository implements Repository on top of an EventStore:
// GetByID rebuilds the aggregate by replaying its stream through
// ApplyEvent; Save appends the uncommitted events with an optimistic
// concurrency check, then publishes them.
type EventStoreRepository[T Aggregate] struct {
	store      EventStore
	factory    func() T
	streamName string
	publisher  DomainEventPublisher
}

// NewEventStoreRepository builds a repository for one aggregate type.
// streamName is the aggregate's stream prefix (e.g. "SalesOrder"), mirroring
// Muflone's aggregateIdToStreamName.
func NewEventStoreRepository[T Aggregate](
	store EventStore,
	factory func() T,
	streamName string,
	publisher DomainEventPublisher,
) *EventStoreRepository[T] {
	return &EventStoreRepository[T]{
		store:      store,
		factory:    factory,
		streamName: streamName,
		publisher:  publisher,
	}
}

func (r *EventStoreRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (T, error) {
	var zero T
	stored, err := r.store.ReadStream(ctx, r.stream(id))
	if err != nil {
		return zero, err
	}
	if len(stored) == 0 {
		return zero, fmt.Errorf("%w: %s/%s", ErrAggregateNotFound, r.streamName, id)
	}
	aggregate := r.factory()
	for _, record := range stored {
		aggregate.ApplyEvent(record.Event)
	}
	return aggregate, nil
}

func (r *EventStoreRepository[T]) Save(ctx context.Context, aggregate T, commitID uuid.UUID) error {
	uncommitted := aggregate.UncommittedEvents()
	if len(uncommitted) == 0 {
		return nil
	}
	expectedVersion := aggregate.Version() - len(uncommitted)
	if err := r.store.Append(ctx, r.stream(aggregate.ID()), expectedVersion, commitID, uncommitted); err != nil {
		return err
	}
	if r.publisher != nil {
		for _, event := range uncommitted {
			if err := r.publisher.PublishDomainEvent(ctx, event); err != nil {
				return err
			}
		}
	}
	aggregate.ClearUncommittedEvents()
	return nil
}

func (r *EventStoreRepository[T]) stream(id uuid.UUID) string {
	return r.streamName + "-" + id.String()
}
