# ADR-0004: Atlas for migrations; no foreign keys across module boundaries

- **Status**: accepted
- **Date**: 2026-07-05

## Context

Schema changes need to be versioned, reviewable, and hash-verified.
Separately, the DDD model treats references to another module's aggregate
as opaque ids, and the database schema should not be stricter than the
model it persists.

## Decision

- Migrations live in `migrations/versions` with `atlas.sum` integrity
  hashes; `migrations/atlas.hcl` defines `local`/`dev`/`prod` envs. The
  compose `migrate` service (image `arigaio/atlas`) applies them before
  the API starts.
- **No FK constraints across modules**: `availabilities.beer_id` and
  `sales_order_rows.beer_id` are logical references only. The single FK
  (`sales_order_rows → sales_orders`) lives inside one aggregate's
  projection.
- The event store (`events` table) is module-agnostic by stream naming
  (`SalesOrder-<id>`, `Availability-<id>`); splitting it per module later
  is a data move, not a DDL redesign.

## Consequences

- Any module's tables can move to a dedicated schema or database without
  DDL surgery on the others.
- Referential integrity across modules is owned by the application layer
  and the event flow — same as it would be between microservices.
- `task migrate:diff -- <name>` generates new versions; `task migrate:hash`
  repairs `atlas.sum` after manual edits.
