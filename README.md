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
- **API hardening & operability** — idempotent order creation (supply your
  own `id` and retry safely), paginated lists (`?limit=&offset=`, enveloped
  as `{items, limit, offset}`), per-IP rate limiting and body-size caps,
  Prometheus metrics at `/metrics`, OTel tracing across HTTP **and bus
  hops** when `OTEL_EXPORTER_OTLP_ENDPOINT` is set, and `task rebuild` to
  reconstruct every read model from the event store.
- **Facades are the only public surface** — the REST layer (Gin) talks to
  `sales.Facade` / `warehouses.Facade`, never to module internals.
- **Authentication & RBAC (Keycloak)** — with `AUTH_ISSUER` set (compose does),
  every `/v1` route requires an OIDC bearer token, and roles gate the writes:
  `viewer` reads, `sales-manager` places orders, `warehouse-operator` declares
  production. SSO comes from the IdP; the realm (roles, test users
  `manager`/`operator`/`barfly`, password `brewup`) is imported from
  `docker/keycloak/realm.json` at boot. Empty `AUTH_ISSUER` = open API (dev).
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

**3. Docker Compose** (full stack: Postgres + RabbitMQ + Keycloak → Atlas migrations → API → demo data):

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
(empty = in-process bus, AMQP URL = RabbitMQ), `AUTH_ISSUER`/`AUTH_JWKS_URL`/
`AUTH_CLIENT_ID` (empty issuer = open API, OIDC issuer = tokens + RBAC),
`SAGA_STEP_TIMEOUT`.

## Using the app

Produce beer, sell it, watch the stock move. In compose mode, grab a token
first (skip this and the `-H` headers when running without auth):

```bash
token() { curl -s localhost:8180/realms/brewup/protocol/openid-connect/token \
  -d grant_type=password -d client_id=brewup-api \
  -d username=$1 -d password=brewup | python3 -c 'import sys,json;print(json.load(sys.stdin)["access_token"])'; }
OPERATOR=$(token operator); MANAGER=$(token manager)

BEER=11111111-1111-1111-1111-111111111111

# 1. A production order fills the warehouse (warehouse-operator role)
curl localhost:8080/v1/warehouses/availability -H "Authorization: Bearer $OPERATOR" -d '{
  "beer_id":"'$BEER'", "beer_name":"BrewUp IPA",
  "quantity":{"value":100,"unit_of_measure":"Lt"}}'

# 2. A customer places a sales order (sales-manager role)
curl localhost:8080/v1/sales -H "Authorization: Bearer $MANAGER" -d '{
  "sales_order_number":"20260705-0001", "customer_name":"Muflone",
  "rows":[{"beer_id":"'$BEER'","beer_name":"BrewUp IPA",
           "quantity":{"value":30,"unit_of_measure":"Lt"},
           "price":{"value":5,"currency":"EUR"}}]}'

# 3. Read models (any authenticated viewer; eventually consistent)
curl localhost:8080/v1/sales -H "Authorization: Bearer $MANAGER"
curl localhost:8080/v1/warehouses/availability -H "Authorization: Bearer $OPERATOR"

```

### API

| Method & path | Purpose |
|---|---|
| `POST /v1/sales` | Place a sales order (returns the new order id) |
| `GET /v1/sales` · `GET /v1/sales/:id` | Query order projections (lists paginate: `?limit=&offset=`) |
| `POST /v1/warehouses/availability` | Declare a finished production order |
| `GET /v1/warehouses/availability[/:beerId]` | Query stock projections |
| `GET /healthz` · `GET /readyz` · `GET /metrics` | Liveness · readiness · Prometheus (open, no token needed) |

RBAC: `POST /v1/sales` needs `sales-manager`; `POST /v1/warehouses/availability`
needs `warehouse-operator`; `GET`s need `viewer`. Keycloak admin console:
http://localhost:8180 (admin/admin).

## Development

```bash
task lint                 # gofmt + go vet
task check:architecture   # fitness functions (module boundaries)
task test:race            # all tests under the race detector
task cover                # hard gate: 100% unit coverage
task migrate:apply        # Atlas migrations (see migrations/)
```

CI (`.github/workflows/ci.yml`) runs the same gate on every push/PR, plus
vulnerability scanning (govulncheck + Trivy, scan-only), migration
verification against Postgres, and a compose smoke test that builds and
boots the production image.
