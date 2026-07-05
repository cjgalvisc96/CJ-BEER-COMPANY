# ADR-0004: Atlas for migrations; no foreign keys across context boundaries

- **Status**: accepted
- **Date**: 2026-07-05

## Context

Schema changes need to be versioned, reviewable, and hash-verified (the
reference platform already standardizes on Atlas). Separately: the DDD
model treats references to another context's aggregate as *opaque IDs*
(e.g. `inventory.StockItem.beerID`), and the database schema should not be
stricter than the model it persists.

## Decision

- Migrations live in `migrations/versions` with `atlas.sum` integrity
  hashes; `migrations/atlas.hcl` defines `local`/`dev`/`prod` envs. The
  compose `migrate` service (image `arigaio/atlas`) applies them before the
  API starts.
- **No FK constraints across contexts**: `stock_items.beer_id`,
  `batches.beer_id`, `order_lines.beer_id` reference the catalog only
  logically. The single FK in the schema (`order_lines → orders`) is inside
  one aggregate boundary.

## Consequences

- Any context's tables can move to a dedicated schema or database without
  DDL surgery on the others.
- Referential integrity across contexts is owned by the application layer
  (ports validate beers exist before use) and by events — same as it would
  be between microservices.
- `task migrate:diff -- <name>` generates new versions against a throwaway
  dev database (`docker://postgres/16/dev`); `task migrate:hash` repairs
  `atlas.sum` after manual edits.
