# ADR-0005: CQRS + Event Sourcing following the BrewUp reference (book Ch. 7)

- **Status**: accepted (supersedes the CRUD-style write model of ADR-0003)
- **Date**: 2026-07-05

## Context

The project's stated goal is to follow *Domain-Driven Refactoring* as
closely as possible in Go. The book's end state for the BrewUp monolith is
CQRS **with event sourcing**: aggregates raise domain events as their only
state changes, an event store is the source of truth, and read models are
projections updated by event handlers.

## Decision

- `internal/muflone` reimplements the essentials of the authors' Muflone
  library: `Command`/`DomainEvent` with aggregateId + commitId,
  `AggregateRoot` (RaiseEvent/Apply/Version), `EventStoreRepository` with
  stream replay and optimistic concurrency, a `ServiceBus`
  (producer-consumer for commands, pub/sub for events), and the
  `CommandSpecification` Given/When/Expect harness.
- Each module (`sales`, `warehouses`) mirrors the BrewUp project split:
  SharedKernel (customtypes/commands/events/integrationevents), Domain
  (aggregate + command handlers), ReadModel (dtos/eventhandlers/services),
  Facade.
- Writes return the pre-generated aggregate id immediately; reads are
  eventually consistent projections (202/201 + poll the read model).
- Architecture fitness tests (`tests/architecture_test.go`) enforce module
  isolation and the REST-only-through-facades rule, replacing the book's
  NetArchTest examples.

## Consequences

- Full auditability: the event streams are "the movie of the aggregate,
  not just the picture".
- Eventual consistency between write and read sides is now explicit in the
  API contract and in every e2e test (`require.Eventually`).
- The in-memory event store is swappable: the Postgres schema for streams
  and projections is versioned in `migrations/` (an `events` table plus
  projection tables), so moving to a durable store is an adapter, not a
  redesign.
- Event versioning/upcasting (book Ch. 11) and sagas with compensation
  (Ch. 12) remain future ADRs — the current flow is a simple choreography.
