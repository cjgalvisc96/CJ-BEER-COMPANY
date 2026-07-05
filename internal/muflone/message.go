// Package muflone is a small Go homage to Muflone
// (https://github.com/CQRS-Muflone/Muflone), the open source CQRS + Event
// Sourcing library used throughout the book Domain-Driven Refactoring for
// the BrewUp application. It provides the same building blocks: commands
// and domain events carrying an aggregateId and a commitId, an
// AggregateRoot that raises and applies events, an event-store-backed
// repository with optimistic concurrency, a service bus, and a
// specification-test harness.
package muflone

import "github.com/google/uuid"

// Message is anything that travels on the service bus. MessageName is the
// stable wire name (and topic suffix) of the message.
type Message interface {
	MessageName() string
}

// Command is a request to change the state of one aggregate. Commands are
// named in the imperative form (CreateSalesOrder) and are delivered to
// exactly one handler (producer-consumer pattern).
type Command interface {
	Message
	AggregateID() uuid.UUID
	CommitID() uuid.UUID
}

// CommandBase carries the identity every command shares: the target
// aggregate and the commitId correlating the command with the events it
// produces. Embed it in concrete commands.
type CommandBase struct {
	AggregateId uuid.UUID `json:"aggregate_id"`
	CommitId    uuid.UUID `json:"commit_id"`
}

func NewCommandBase(aggregateID, commitID uuid.UUID) CommandBase {
	return CommandBase{AggregateId: aggregateID, CommitId: commitID}
}

func (c CommandBase) AggregateID() uuid.UUID { return c.AggregateId }
func (c CommandBase) CommitID() uuid.UUID    { return c.CommitId }

// DomainEvent is a fact that happened inside one bounded context, named in
// the past tense (SalesOrderCreated). Domain events stay within their
// context; sharing across contexts happens through integration events.
type DomainEvent interface {
	Message
	AggregateID() uuid.UUID
	CommitID() uuid.UUID
}

// DomainEventBase mirrors Muflone's DomainEvent(aggregateId, commitId).
type DomainEventBase struct {
	AggregateId uuid.UUID `json:"aggregate_id"`
	CommitId    uuid.UUID `json:"commit_id"`
}

func NewDomainEventBase(aggregateID, commitID uuid.UUID) DomainEventBase {
	return DomainEventBase{AggregateId: aggregateID, CommitId: commitID}
}

func (e DomainEventBase) AggregateID() uuid.UUID { return e.AggregateId }
func (e DomainEventBase) CommitID() uuid.UUID    { return e.CommitId }

// IntegrationEvent is designed to be shared across bounded contexts (or
// systems): it carries only the essential data other contexts need,
// keeping them loosely coupled.
type IntegrationEvent interface {
	Message
}

// IntegrationEventBase keeps the commitId so a cross-context reaction can
// be correlated back to the command that caused it.
type IntegrationEventBase struct {
	CommitId uuid.UUID `json:"commit_id"`
}

func NewIntegrationEventBase(commitID uuid.UUID) IntegrationEventBase {
	return IntegrationEventBase{CommitId: commitID}
}
