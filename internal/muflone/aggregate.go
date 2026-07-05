package muflone

import "github.com/google/uuid"

// Aggregate is what the repository knows how to load and save. Concrete
// aggregates embed AggregateRoot and register their event router.
type Aggregate interface {
	ID() uuid.UUID
	Version() int
	ApplyEvent(event DomainEvent)
	UncommittedEvents() []DomainEvent
	ClearUncommittedEvents()
}

// EventRouter dispatches an event to the aggregate's private apply method
// (the equivalent of Muflone's RegisteredRoutes.Dispatch).
type EventRouter interface {
	Route(event DomainEvent)
}

// AggregateRoot is the event-sourced aggregate base: state changes are
// expressed only as domain events. RaiseEvent applies the event to the
// in-memory state AND records it as uncommitted; the repository persists
// the uncommitted events and the same ApplyEvent path rebuilds the
// aggregate when replaying its stream.
type AggregateRoot struct {
	id          uuid.UUID
	version     int
	uncommitted []DomainEvent
	router      EventRouter
}

// Bind wires the concrete aggregate's router. Call it in the aggregate's
// constructor before any event is raised or applied.
func (a *AggregateRoot) Bind(router EventRouter) {
	a.router = router
}

// SetID is called from the aggregate's apply methods when the creation
// event assigns the identity.
func (a *AggregateRoot) SetID(id uuid.UUID) {
	a.id = id
}

func (a *AggregateRoot) ID() uuid.UUID { return a.id }
func (a *AggregateRoot) Version() int  { return a.version }

// RaiseEvent records a new fact: it is applied to the state and queued for
// persistence.
func (a *AggregateRoot) RaiseEvent(event DomainEvent) {
	a.ApplyEvent(event)
	a.uncommitted = append(a.uncommitted, event)
}

// ApplyEvent mutates state through the router and bumps the version. The
// repository calls this for every event read from the stream.
func (a *AggregateRoot) ApplyEvent(event DomainEvent) {
	a.router.Route(event)
	a.version++
}

func (a *AggregateRoot) UncommittedEvents() []DomainEvent {
	return a.uncommitted
}

func (a *AggregateRoot) ClearUncommittedEvents() {
	a.uncommitted = nil
}
