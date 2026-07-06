# Architecture Overview

CJ Beer Company is the Go rendition of **BrewUp**, the brewery application
built throughout *Domain-Driven Refactoring* (Colla & Acerbis), at the end
state of the book's refactoring journey: a **modular monolith with CQRS and
event sourcing** (the book's `03-monolith_with_cqrs_and_event_sourcing`).

> New to modular monoliths, CQRS, event sourcing or sagas? Read
> [concepts.md](concepts.md) first — it explains every pattern named on this
> page in plain English. This document assumes those terms are familiar.

## Modules (bounded contexts)

| Module | Aggregate | Responsibility |
|---|---|---|
| `sales` | `SalesOrder` | Customer orders: `CreateSalesOrder` → `SalesOrderCreated` |
| `warehouses` | `Availability` | Beer stock: production fills it, sales orders allocate from it |

`Payment` and `Shipping` complete the book's context map; they are
described in the event flow (Chapter 4, Figure 4.2) but — like in the
book's code — not implemented yet.

## Module layout (mirrors the BrewUp projects)

```
internal/<module>/
├── sharedkernel/         BrewUp.<Module>.SharedKernel — the module's published language
│   ├── customtypes.go       strongly named ids/values (SalesOrderId, BeerName, …)
│   ├── commands/            imperative messages (CreateSalesOrder)
│   ├── events/              past-tense domain events (SalesOrderCreated)
│   └── integrationevents/   events shared with OTHER contexts (separate types!)
├── domain/               BrewUp.<Module>.Domain — the event-sourced write model
│   ├── <aggregate>.go       RaiseEvent/apply pairs; state changes only via events
│   └── commandhandlers/     one handler per command; load/create aggregate → Save
├── readmodel/            BrewUp.<Module>.ReadModel — the query side
│   ├── dtos/                query-only shapes, no domain behavior
│   ├── eventhandlers/       projections subscribing to the module's domain events
│   └── services/            projection writers + query services
├── facade.go             ISalesFacade / IWarehousesFacade — the ONLY public surface
└── module.go             composition root of the module
```

`internal/rest` is BrewUp.Rest: it maps `/v1/sales` and `/v1/warehouses`
endpoints and depends **only on facades** — enforced by the architecture
fitness tests in `tests/architecture_test.go` (the NetArchTest equivalent).

## The muflone package

`internal/muflone` is a small Go homage to
[Muflone](https://github.com/CQRS-Muflone/Muflone), the authors' CQRS/ES
library used by BrewUp. It provides:

- `Command` / `DomainEvent` — messages carrying `aggregateId` + `commitId`
- `IntegrationEvent` — cross-context messages (kept distinct from domain events)
- `AggregateRoot` — `RaiseEvent` applies the event and queues it as
  uncommitted; `ApplyEvent` routes to the aggregate's apply methods and
  bumps `Version`
- `EventStoreRepository` — `GetByID` replays the stream; `Save` appends
  uncommitted events with optimistic concurrency. In durable mode it also
  writes each event to an `outbox` table **in the same transaction**, and the
  `OutboxRelay` publishes from there (no crash can lose a message — ADR-0012);
  in-memory mode publishes directly
- Two `EventStore` adapters — `InMemoryEventStore` (dev, tests) and
  `PostgresEventStore` (production: the `events` table in `migrations/`,
  concurrency via head check + primary key). `DB_URL` selects the mode
  (ADR-0006)
- `EventRegistry` — rehydrates stored events by type name, with
  **upcasters** for schema evolution (ADR-0007, book Ch. 11)
- `ServiceBus` — commands via producer-consumer (one handler), events via
  pub/sub, on a pluggable `Transport`: in-process GoChannel by default,
  **RabbitMQ** (the book's broker) via `BROKER_URL` — durable queue per
  handler, fan-out on shared topics, competing consumers across replicas
  (ADR-0009). Failing handlers are retried with backoff, then the message
  is parked on the `brewup.dead_letter` topic (never lost, never blocking)
- `CommandSpecification` — the Given/When/Expect test harness

## How a request flows (CQRS + ES)

```
POST /v1/sales
  → Facade builds CreateSalesOrder (new SalesOrderId, commitId) → bus.Send
  → CreateSalesOrderCommandHandler → SalesOrder factory checks invariants
      → RaiseEvent(SalesOrderCreated)
  → Repository.Save appends to the SalesOrder-<id> stream (event store)
      → durable mode: an outbox row is written in the SAME transaction; the
        relay publishes SalesOrderCreated (in-memory mode publishes directly)
  → readmodel: SalesOrderCreatedEventHandler projects the SalesOrder DTO
  → integration: sales republishes SalesOrderCreated as an INTEGRATION event
  → warehouses: consumes it (consumer-driven contract), sends
      UpdateAvailabilityDueToSalesOrder per row
  → Availability raises BeerAvailabilityUpdated (remaining quantity)
      → projected into the availability read model
      → republished as integration event → Sales is notified
GET /v1/sales, /v1/warehouses/availability   ← read models (eventually consistent)
```

## Production & operability

The same binary runs zero-dependency for dev/tests and fully durable in
production; environment variables flip each concern independently (see
[concepts.md → Running it in production](concepts.md#running-it-in-production)):

| Concern | Off (empty env) | On | Where | ADR |
|---|---|---|---|---|
| Persistence | in-memory | Postgres event store + read models (`DB_URL`) | `muflone`, `internal/platform/database` | 0006 |
| Messaging | in-process bus | RabbitMQ, durable + multi-replica (`BROKER_URL`) | `muflone` transport | 0009 |
| At-least-once delivery | direct publish | transactional outbox + relay (`OUTBOX_INTERVAL`) | `muflone/outbox.go` | 0012 |
| Reliability | retry then log | concurrency-aware retries → dead-letter archive → `task redrive` | `muflone/retry.go`, `deadletters.go` | 0012 |
| Auth | open API | OIDC bearer tokens + RBAC (`AUTH_ISSUER`) | `internal/platform/auth`, `internal/rest/auth.go` | 0010 |
| Tracing | metrics only | OTLP traces across HTTP + bus hops (`OTEL_EXPORTER_OTLP_ENDPOINT`) | `internal/platform/telemetry` | 0011 |
| Edge hardening | always on | idempotency, pagination, rate limit, body cap, trusted proxies | `internal/rest` | 0011 |

Everything is tagged with `APP_ENV` (`local`/`dev`/`staging`/`prod`): it labels
every log line and the OpenTelemetry `deployment.environment` on traces and
metrics, so a shared backend never confuses environments.

## Rules enforced by construction and by fitness tests

1. Modules never import each other — cross-context communication is
   integration events on the bus, deserialized into consumer-owned
   contract structs (topics are the only shared contract).
2. Domain events stay inside their module; integration events are separate
   types even when the shape matches (Chapter 4's warning).
3. REST depends only on facades.
4. `muflone` knows nothing about the business modules.
5. Everything under `internal/` — compiler-enforced privacy.
