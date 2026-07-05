# ADR-0002: Cross-context collaboration via event choreography (Watermill)

- **Status**: accepted
- **Date**: 2026-07-05

## Context

Inventory must react to production (brewing) and sales (orders). Direct
calls between contexts' write sides would couple their transactions and
their availability.

## Decision

Contexts communicate through events on a Watermill bus (GoChannel pub/sub
in-process today). The reservation flow is a *choreography*: orders publishes
`order_placed`; inventory reserves and publishes the outcome
(`order_stock_reserved|rejected`); orders settles the order. Consumers
define their own contract structs; topic names are the only shared surface.
Read-side lookups that must be synchronous (order placement validating a
beer) go through an application-layer port + ACL adapter instead.

## Consequences

- Placing an order returns **202** with a `pending` order; confirmation is
  eventually consistent (tests poll with `require.Eventually`).
- Swapping GoChannel for Kafka/NATS touches only
  `shared/infrastructure/messaging`.
- No saga orchestrator or compensation yet — acceptable while reservations
  are single-consumer and in-process (revisit with ADR when extracting a
  context).
