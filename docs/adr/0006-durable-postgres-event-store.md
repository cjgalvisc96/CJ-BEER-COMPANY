# ADR-0006: Durable Postgres event store and projections (book Ch. 8)

- **Status**: accepted (supersedes the "in-memory only" stance of ADR-0003)
- **Date**: 2026-07-05

## Context

The book's journey ends production-ready: Chapter 8 covers refactoring the
database for the modular, event-driven architecture. An in-memory event
store validates the model but loses everything on restart — not
production.

## Decision

- `muflone.PostgresEventStore` implements the `EventStore` port over the
  `events` table (append-only streams keyed by `(stream_id, version)`).
  Optimistic concurrency is enforced by a head check inside the
  transaction plus the primary key itself (unique violation →
  `ErrConcurrency`).
- Each module's read model gets a Postgres adapter
  (`PostgresSalesOrderService`, `PostgresAvailabilityService`) over the
  projection tables.
- **`DB_URL` selects the mode**: empty → in-memory (dev, tests); a
  Postgres URL → durable (the compose stack and production). The switch
  lives in each module's `Register` — domain, handlers, and facades are
  identical in both modes.
- `/readyz` reports database reachability; the app fails fast at boot if
  `DB_URL` is set but unreachable.

## Consequences

- State survives restarts (proven by the CI smoke test, which restarts the
  API mid-flow and asserts the read models are intact).
- The adapters are unit-tested to 100% with sqlmock (every error branch)
  and integration-tested against real Postgres by the compose smoke test.
- The event store table is shared across modules but partitioned by stream
  naming; per-module databases remain a data move (ADR-0004).
- Read-model queries now return errors honestly (`muflone.ErrNotFound` →
  404); a database outage surfaces as 500/`/readyz` failing, not silence.
