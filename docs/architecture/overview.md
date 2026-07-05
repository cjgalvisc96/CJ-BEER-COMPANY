# Architecture Overview

## Bounded contexts

| Context | Aggregate | Responsibility |
|---|---|---|
| `catalog` | `Beer` | What we sell: styles, ABV, price, active/retired lifecycle |
| `brewing` | `Batch` | What we produce: production runs from start to completion |
| `inventory` | `StockItem` | What we have: stock levels, replenishment, reservation |
| `orders` | `Order` | What we sold: order lifecycle pending → confirmed/rejected/cancelled |
| `shared` | — | Shared kernel: `EntityID`, `Money`, `AggregateRoot`, `Event`, error kinds |

## Layering (inside every context)

```
domain/           pure model: aggregates, VOs, events, repository PORTS, errors
   ▲
application/      use cases: commands/, queries/, dto/, ports/ (to other contexts),
   ▲              eventhandlers/ (reactions to other contexts' events)
infrastructure/   adapters: persistence/ (repositories), acl/ (other-context ports)
module.go         per-context DI wiring — the only cross-layer file
```

Dependency rules (all enforced by the Go compiler where possible):

1. `domain` imports only `internal/shared/domain`.
2. `application` imports its own `domain` + shared ports.
3. `infrastructure` implements ports; nothing imports infrastructure except
   the context's `module.go`.
4. **Cross-context**: only `infrastructure/acl` may import another context,
   and only that context's *application* layer (never its domain).
5. `presentation/http` sees application DTOs and handlers only.
6. Everything lives under `internal/` — the Go compiler forbids any other
   module from importing it (the toolchain-native version of an
   import-linter contract).

## Composition root

`internal/app/app.go` builds the samber/do injector, registers the shared
services (logger, config, Watermill bus, event publisher), calls each
context's `Register`, subscribes event handlers, and assembles the Gin
router. `cmd/api/main.go` is a thin shell: load config → `app.New` →
`app.Run` with signal-driven graceful shutdown.

## Persistence

Repositories are domain-owned interfaces. Today's adapters are in-memory
(snapshot-based, race-safe); the target schema for the Postgres adapters is
version-controlled in [`migrations/`](../../migrations) and managed with
Atlas. Deliberately, the schema has **no cross-context foreign keys** — see
[ADR-0004](../adr/0004-atlas-migrations-no-cross-context-fks.md).
