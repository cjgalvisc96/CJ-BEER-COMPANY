# ADR-0003: In-memory repositories first, Postgres schema versioned from day one

- **Status**: accepted
- **Date**: 2026-07-05

## Context

The domain model and context boundaries carry the project's risk; a database
adds operational weight without validating them. But deferring all
persistence thinking tends to leak "whatever the ORM produced" into the
domain.

## Decision

Ship snapshot-based, race-safe in-memory adapters behind the domain-owned
repository ports, **and** version the target Postgres schema in
`migrations/` (Atlas) from the start, so the persistence contract is
designed deliberately rather than emerging by accident. The compose stack
already brings up Postgres + migrations to keep the path exercised.

## Consequences

- `task run` and the whole test suite need zero external services.
- Data does not survive a restart — fine for the current stage, and the
  seeder (`docker/mock-data/seed.sh`) repopulates demo data.
- Adding the real adapter = one file per context in
  `infrastructure/persistence` + an outbox decision (future ADR); no domain
  or application changes.
