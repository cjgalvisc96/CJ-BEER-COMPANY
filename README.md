# CJ-BEER-COMPANY 🍺

A craft-beer company system built as a **modular monolith** in Go, applying **Domain-Driven Design**
(in the spirit of *Domain-Driven Refactoring*): bounded contexts, strict
`domain → application → infrastructure` layering, CQRS-style use cases, and event-driven
choreography between contexts.

## Stack

| Concern | Library |
|---|---|
| HTTP API | [Gin](https://github.com/gin-gonic/gin) |
| Messaging / event bus | [Watermill](https://github.com/ThreeDotsLabs/watermill) (GoChannel pub/sub) |
| Dependency injection | [samber/do v2](https://github.com/samber/do) |
| IDs | google/uuid |
| Logging | stdlib `log/slog` (JSON) |
| Tests | stretchr/testify |

## Quick start

Requires [Task](https://taskfile.dev) (`task --list` shows every command):

```bash
cp .env.example .env   # optional; sensible defaults are baked in
task run               # zero-dependency: serve on :8080 with in-memory storage
task test              # unit + end-to-end tests
task test:race         # same, with the race detector
task lint              # go vet + gofmt check
```

Or the full stack — ordered bring-up `postgres → atlas migrate → api → seed`:

```bash
task docker:up         # migrated database + API on :8080 + demo data
task docker:down       # stop (add `-- -v` to drop the db volume)
```

Migrations are managed with [Atlas](https://atlasgo.io):
`task migrate:apply`, `task migrate:status`, `task migrate:hash` (all run
via the `arigaio/atlas` image — no local install needed).

### Try the full flow

```bash
# 1. Add a beer to the catalog
BEER=$(curl -s localhost:8080/api/v1/beers -d '{
  "name":"CJ Golden Lager","style":"lager","abv":4.8,
  "price_cents":450,"currency":"USD","description":"Flagship lager"
}' | jq -r .id)

# 2. Brew it: start a batch, then complete it → inventory is replenished by event
BATCH=$(curl -s localhost:8080/api/v1/batches -d "{\"beer_id\":\"$BEER\",\"units\":100}" | jq -r .id)
curl -s localhost:8080/api/v1/batches/$BATCH/complete -d '{"produced_units":95}' | jq
curl -s localhost:8080/api/v1/stock/$BEER | jq          # quantity: 95

# 3. Sell it: place an order → inventory reserves stock → order is confirmed by event
ORDER=$(curl -s localhost:8080/api/v1/orders -d "{
  \"customer_name\":\"Bar La Cerveceria\",
  \"lines\":[{\"beer_id\":\"$BEER\",\"units\":30}]
}" | jq -r .id)
curl -s localhost:8080/api/v1/orders/$ORDER | jq .status  # "confirmed"
curl -s localhost:8080/api/v1/stock/$BEER | jq .quantity  # 65
```

## Architecture

### Bounded contexts

```
internal/
├── catalog/      What we sell: Beer aggregate (style, ABV, price, lifecycle)
├── brewing/      What we produce: Batch aggregate (start → complete)
├── inventory/    What we have: StockItem aggregate (replenish / reserve)
├── orders/       What we sold: Order aggregate (pending → confirmed/rejected/cancelled)
├── shared/       Shared kernel: EntityID, Money, AggregateRoot, Event, error kinds
├── presentation/ Gin handlers + router (talks only to application DTOs)
├── platform/     config, logging
└── app/          Composition root: DI container, bus wiring, graceful shutdown
```

> **Why `internal/` and not `src/`?** In Go the layout *is* the import path:
> packages under `internal/` are un-importable from any other module — a
> compiler-enforced privacy boundary (the toolchain-native equivalent of an
> import-linter contract). `cmd/` + `internal/` is the standard Go layout.

Repo scaffolding around the code:

| Path | Purpose |
|---|---|
| `migrations/` | Atlas-managed Postgres schema (`atlas.hcl`, hashed `versions/`) |
| `docker/` + `docker-compose.yml` | Ordered local stack: postgres → migrate → api → seed |
| `docs/` | Architecture overview, event catalog, ADRs, testing guide |
| `.claude/` | AI-harness rules, skills, and the quality-gate command |
| `.agents/` | Role charters (lead/architect/developer/tester/devops/sre) |
| `.env.example` | Configuration template (compose reads `.env` automatically) |

Each context is a vertical slice with the same internal shape:

```
<context>/
├── domain/          Aggregates, value objects, domain events, repository PORT, errors
├── application/
│   ├── commands/    Write use cases (one handler per file)
│   ├── queries/     Read use cases
│   ├── dto/         Boundary data shapes
│   ├── ports/       Driven ports to OTHER contexts (anti-corruption layer)
│   └── eventhandlers/  Reactions to other contexts' events
├── infrastructure/
│   ├── persistence/ Repository adapters (in-memory today; swap for Postgres here)
│   └── acl/         Adapters implementing the ports against other contexts
└── module.go        Per-context DI wiring (the only cross-layer file)
```

### Event choreography

Contexts never call each other's write side. They collaborate through events on the Watermill bus:

```
brewing                inventory                    orders
   │                       │                           │
   │ batch_completed ────▶ replenish stock             │
   │                       │                           │
   │                       │ ◀──────────── order_placed│
   │                       │ reserve stock             │
   │                       │── order_stock_reserved ──▶│ confirm order
   │                       │── order_stock_rejected ──▶│ reject order
```

Consumers deserialize **their own local contract structs** (only the fields they need) instead of
importing the producer's types — contexts stay independently evolvable, and the topics
(`orders.order_placed`, `brewing.batch_completed`, …) are the only public contract.

### Rules enforced by construction

- **Layer direction**: domain imports nothing but the shared kernel; application imports domain;
  infrastructure implements ports. Presentation only sees application DTOs.
- **Context isolation**: the only file in a context that may import another context is its
  `infrastructure/acl` adapter, and it talks to the other context's *application* layer.
- **Ubiquitous language**: `Beer.Retire()`, `Batch.Complete()`, `StockItem.Reserve()`,
  `Order.Reject()` — behavior lives on the aggregates, not in services (no anemic model).
- **Events after persistence**: aggregates record events; handlers publish them only after a
  successful `Save` (`repository.Save` → `publisher.Publish(aggregate.PullEvents()...)`).
- **Prices are captured at order time** (`OrderLine.unitPrice`): catalog price changes never
  rewrite history.

### API

| Method & path | Purpose |
|---|---|
| `POST /api/v1/beers` · `GET /api/v1/beers[/:id]` | Create / read beers |
| `PUT /api/v1/beers/:id/price` · `DELETE /api/v1/beers/:id` | Change price / retire |
| `POST /api/v1/batches` · `POST /api/v1/batches/:id/complete` | Start / complete a brew |
| `GET /api/v1/batches[/:id]` | Read batches |
| `POST /api/v1/stock` · `POST /api/v1/stock/:beerId/replenish` | Track / replenish stock |
| `GET /api/v1/stock[/:beerId]` | Read stock levels |
| `POST /api/v1/orders` (→ **202**, settles async) · `POST /api/v1/orders/:id/cancel` | Place / cancel orders |
| `GET /api/v1/orders[/:id]` | Read orders |
| `GET /healthz` | Liveness |

Domain errors map to HTTP in one place (`presentation/http/respond.go`):
validation → 400, not found → 404, conflict → 409, business-rule rejection → 422.

## Design notes (YAGNI applied)

- **In-memory repositories** behind domain-owned ports: the demo runs with zero external
  dependencies; a Postgres adapter is a new file in `infrastructure/persistence`, nothing else
  changes.
- **GoChannel pub/sub** for the same reason: Watermill makes Kafka/NATS a config change in
  `shared/infrastructure/messaging`, not a redesign.
- **No sagas/outbox yet**: the reservation flow is a simple choreography; add an outbox when a
  real database arrives.
