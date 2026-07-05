# CJ-BEER-COMPANY 🍺

A brewery system in Go — the Go rendition of **BrewUp**, the application built
throughout *Domain-Driven Refactoring* (Colla & Acerbis): a **modular monolith
with CQRS and event sourcing**.

## Core concepts

- **Two bounded contexts**, isolated modules that never import each other:
  - **`sales`** — customer orders (`CreateSalesOrder` → `SalesOrderCreated`)
  - **`warehouses`** — beer availability (production orders fill it, sales orders allocate from it)
- **CQRS + Event Sourcing** — aggregates change state only by raising domain
  events; an event store (streams + optimistic concurrency) is the source of
  truth; read models are projections you query. `DB_URL` selects persistence:
  empty runs in memory (dev/tests), a Postgres URL makes everything durable
  (production — the compose stack does this, and state survives restarts).
- **Events over calls** — modules collaborate through integration events on a
  service bus (Watermill). The wire is pluggable: in-process by default,
  **RabbitMQ with durable queues** when `BROKER_URL` is set (the compose
  stack does — messages survive restarts, replicas compete for work).
  A sales order crossing to the warehouse:

  ```
  Sales                          Warehouses
    │ SalesOrderCreated ────────────▶ order-allocation SAGA: one step per row
    │ ◀── order_allocation_completed │ (a failed step compensates the
    │     or _rejected               │  already-allocated rows — book Ch. 12)
    order status → allocated | rejected
  ```
- **Saga with compensating transactions** — the allocation is an
  event-sourced saga (`OrderAllocationSaga-<orderId>` stream): a shortage is
  the book's `QuantityNotFound` event, previously allocated rows are given
  back (backward recovery), every message is idempotent under redelivery.
- **Durable execution** — in-flight sagas resume at boot, a watchdog times
  out stalled steps (`SAGA_STEP_TIMEOUT`), and poison messages are retried
  then parked on a dead-letter topic instead of being lost.
- **Facades are the only public surface** — the REST layer (Gin) talks to
  `sales.Facade` / `warehouses.Facade`, never to module internals.
- **`internal/muflone`** — a small Go homage to the authors'
  [Muflone](https://github.com/CQRS-Muflone/Muflone) library: commands/events
  with `aggregateId` + `commitId`, `AggregateRoot` (RaiseEvent/Apply),
  event-store repository, service bus, and a Given/When/Expect
  specification-test harness.
- **Architecture enforced by tests** — fitness functions
  (`task check:architecture`) fail the build on any forbidden dependency, and
  a hard gate requires **100% unit-test coverage** (`task cover`).

Deep dives: [`docs/`](docs/) (architecture overview, message catalog, ADRs).

## Installation

Pick one:

**1. Go only** (zero dependencies, everything in memory):

```bash
go run ./cmd/api                # serves on :8080
```

**2. With [Task](https://taskfile.dev)** (same, plus the dev workflow):

```bash
task run                        # run the API
task test                       # all tests
task --list                     # everything else
```

**3. Docker Compose** (full stack: Postgres → Atlas migrations → API → demo data):

```bash
task docker:up                  # or: docker compose up --build -d
task docker:down                # stop (add `-- -v` to drop the db volume)
```

**4. Docker image** (production distroless build):

```bash
docker build -t cj-beer-company .          # or: task docker:build
docker run -p 8080:8080 cj-beer-company
```

Configuration via environment (see `.env.example`): `HTTP_ADDR`, `LOG_LEVEL`,
`GIN_MODE`, `DB_URL` (empty = in-memory, Postgres URL = durable), `BROKER_URL`
(empty = in-process bus, AMQP URL = RabbitMQ), `SAGA_STEP_TIMEOUT`.

## Using the app

Produce beer, sell it, watch the stock move:

```bash
BEER=11111111-1111-1111-1111-111111111111

# 1. A production order fills the warehouse
curl localhost:8080/v1/warehouses/availability -d '{
  "beer_id":"'$BEER'", "beer_name":"BrewUp IPA",
  "quantity":{"value":100,"unit_of_measure":"Lt"}}'

# 2. A customer places a sales order
curl localhost:8080/v1/sales -d '{
  "sales_order_number":"20260705-0001", "customer_name":"Muflone",
  "rows":[{"beer_id":"'$BEER'","beer_name":"BrewUp IPA",
           "quantity":{"value":30,"unit_of_measure":"Lt"},
           "price":{"value":5,"currency":"EUR"}}]}'

# 3. Read models (eventually consistent — writes return immediately)
curl localhost:8080/v1/sales                          # projected orders
curl localhost:8080/v1/warehouses/availability        # 70 Lt remaining
```

### API

| Method & path | Purpose |
|---|---|
| `POST /v1/sales` | Place a sales order (returns the new order id) |
| `GET /v1/sales` · `GET /v1/sales/:id` | Query order projections |
| `POST /v1/warehouses/availability` | Declare a finished production order |
| `GET /v1/warehouses/availability[/:beerId]` | Query stock projections |
| `GET /healthz` · `GET /readyz` | Liveness · readiness (checks the database in durable mode) |

## Development

```bash
task lint                 # gofmt + go vet
task check:architecture   # fitness functions (module boundaries)
task test:race            # all tests under the race detector
task cover                # hard gate: 100% unit coverage
task migrate:apply        # Atlas migrations (see migrations/)
```

CI (`.github/workflows/ci.yml`) runs the same gate on every push/PR, plus
vulnerability scanning, migration verification against Postgres, a compose
smoke test, and the image build/scan/publish to GHCR.
