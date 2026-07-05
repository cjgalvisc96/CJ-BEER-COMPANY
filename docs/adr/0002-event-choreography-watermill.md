# ADR-0002: Cross-context collaboration via events on a service bus (Watermill)

- **Status**: accepted (refined by ADR-0005)
- **Date**: 2026-07-05

## Context

Modules must react to each other (a sales order allocates warehouse stock)
without coupling their write sides. The book's journey replaces the
mediator (direct, synchronous orchestration between facades) with a
service bus and events.

## Decision

Messages travel on a service bus (`muflone.ServiceBus`, Watermill GoChannel
in-process today; the book uses RabbitMQ — a transport detail):

- **Commands**: producer-consumer, exactly one handler.
- **Domain events**: pub/sub inside the owning module (projections,
  integration publishers).
- **Integration events**: pub/sub across modules; consumers deserialize
  their own contract structs and never import the producer's types. The
  flow is a *choreography* — no central orchestrator.

## Consequences

- Writing returns immediately with the pre-generated aggregate id;
  outcomes are eventually consistent (tests poll with `require.Eventually`).
- Swapping GoChannel for RabbitMQ/Kafka touches only the ServiceBus.
- Compensation logic arrived with the order-allocation process: see
  ADR-0008 for the event-sourced saga with compensating transactions
  (book Ch. 12) that coordinates it.
