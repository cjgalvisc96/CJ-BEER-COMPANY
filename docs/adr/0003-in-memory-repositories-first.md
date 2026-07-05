# ADR-0003: In-memory persistence first, target schema versioned from day one

- **Status**: accepted (write model superseded by ADR-0005's event store)
- **Date**: 2026-07-05

## Context

The domain model and module boundaries carry the project's risk; a
database adds operational weight without validating them.

## Decision

Run everything in memory — the event store (`muflone.InMemoryEventStore`)
and the read-model services — while versioning the target Postgres schema
in `migrations/` (Atlas) from the start: an `events` table for the streams
and projection tables for the read models. The compose stack already
brings up Postgres + migrations so the path stays exercised.

## Consequences

- `task run` and the whole test suite need zero external services.
- Data does not survive a restart; the seeder repopulates demo data.
- Going durable = implementing `muflone.EventStore` over Postgres/
  EventStoreDB and pointing the read-model services at the projection
  tables; no domain, handler, or facade changes.
