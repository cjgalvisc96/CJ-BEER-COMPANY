# Architecture Overview

CJ Beer Company is the Go rendition of **BrewUp**, the brewery application
built throughout *Domain-Driven Refactoring* (Colla & Acerbis), at the end
state of the book's refactoring journey: a **modular monolith with CQRS and
event sourcing** (the book's `03-monolith_with_cqrs_and_event_sourcing`).

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
  uncommitted events with optimistic concurrency, then publishes them
- `InMemoryEventStore` — streams in memory; source of truth today (swap
  for EventStoreDB/Postgres by implementing `EventStore`; the target SQL
  schema is already in `migrations/`)
- `ServiceBus` — commands via producer-consumer (one handler), events via
  pub/sub, on Watermill (the book uses RabbitMQ; a transport detail)
- `CommandSpecification` — the Given/When/Expect test harness

## How a request flows (CQRS + ES)

```
POST /v1/sales
  → Facade builds CreateSalesOrder (new SalesOrderId, commitId) → bus.Send
  → CreateSalesOrderCommandHandler → SalesOrder factory checks invariants
      → RaiseEvent(SalesOrderCreated)
  → Repository.Save appends to the SalesOrder-<id> stream (event store)
      → bus publishes SalesOrderCreated
  → readmodel: SalesOrderCreatedEventHandler projects the SalesOrder DTO
  → integration: sales republishes SalesOrderCreated as an INTEGRATION event
  → warehouses: consumes it (consumer-driven contract), sends
      UpdateAvailabilityDueToSalesOrder per row
  → Availability raises BeerAvailabilityUpdated (remaining quantity)
      → projected into the availability read model
      → republished as integration event → Sales is notified
GET /v1/sales, /v1/warehouses/availability   ← read models (eventually consistent)
```

## Rules enforced by construction and by fitness tests

1. Modules never import each other — cross-context communication is
   integration events on the bus, deserialized into consumer-owned
   contract structs (topics are the only shared contract).
2. Domain events stay inside their module; integration events are separate
   types even when the shape matches (Chapter 4's warning).
3. REST depends only on facades.
4. `muflone` knows nothing about the business modules.
5. Everything under `internal/` — compiler-enforced privacy.
