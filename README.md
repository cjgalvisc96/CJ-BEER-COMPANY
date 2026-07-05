# CJ-BEER-COMPANY 🍺

The **Go rendition of BrewUp** — the brewery application built throughout
*Domain-Driven Refactoring* (Colla & Acerbis, Packt) — at the end state of
the book's journey: a **modular monolith with CQRS and event sourcing**
(the book's `03-monolith_with_cqrs_and_event_sourcing`, in Go instead of C#).

## Stack

| Concern | Here | In the book |
|---|---|---|
| CQRS/ES building blocks | `internal/muflone` (Go homage) | [Muflone](https://github.com/CQRS-Muflone/Muflone) |
| Service bus | [Watermill](https://github.com/ThreeDotsLabs/watermill) (GoChannel) | RabbitMQ |
| HTTP API | [Gin](https://github.com/gin-gonic/gin) | ASP.NET minimal APIs |
| Dependency injection | [samber/do v2](https://github.com/samber/do) | .NET DI |
| Event store | in-memory (Postgres schema versioned with Atlas) | EventStoreDB |
| Architecture fitness tests | `go/parser` import checks | NetArchTest |
| Tests | testify + `muflone.CommandSpecification` | xUnit + Muflone.SpecificationTests |

## Quick start

Requires [Task](https://taskfile.dev):

```bash
task run          # zero-dependency: serve on :8080, everything in memory
task test         # specification + e2e + architecture fitness tests
task test:race    # same, under the race detector
task cover        # HARD GATE: 100% unit coverage (internal/app exempt)
task docker:up    # postgres → atlas migrate → api → seeded demo data
```

CI (`.github/workflows/ci.yml`) runs the same gate on every push/PR, in
seven jobs: **lint** (gofmt, vet, staticcheck, tidy check),
**architecture** (`task check:architecture` — the fitness functions:
Sales ⊥ Warehouses, REST-through-facades-only, muflone stays generic,
per-module SharedKernel/Domain/ReadModel layering), **test** (race
detector + the 100% coverage gate, coverage artifact), **vulnerabilities**
(govulncheck), **migrations** (`atlas.sum` integrity + real apply against
a Postgres service), **e2e-smoke** (the full compose stack exercising the
production → order → allocation choreography), and **image** (buildx +
Trivy scan; pushed to GHCR with sha/semver/latest tags outside PRs).
Dependabot keeps modules, actions, and base images fresh.

### Try the flow (the book's Figure 4.2)

```bash
BEER=11111111-1111-1111-1111-111111111111

# 1. A production order fills the warehouse
curl -s localhost:8080/v1/warehouses/availability -d "{
  \"beer_id\":\"$BEER\",\"beer_name\":\"BrewUp IPA\",
  \"quantity\":{\"value\":100,\"unit_of_measure\":\"Lt\"}}" | jq

# 2. A customer places a sales order
curl -s localhost:8080/v1/sales -d "{
  \"sales_order_number\":\"20260705-0001\",\"customer_name\":\"Muflone\",
  \"rows\":[{\"beer_id\":\"$BEER\",\"beer_name\":\"BrewUp IPA\",
    \"quantity\":{\"value\":30,\"unit_of_measure\":\"Lt\"},
    \"price\":{\"value\":5,\"currency\":\"EUR\"}}]}" | jq

# 3. Eventually consistent read models
curl -s localhost:8080/v1/sales | jq                      # the projected order
curl -s localhost:8080/v1/warehouses/availability | jq    # 70 Lt remaining
```

## Architecture

Two modules — `sales` and `warehouses` — exactly as BrewUp codes them,
each split like the book's projects:

```
internal/
├── muflone/            Command/DomainEvent (aggregateId + commitId), AggregateRoot
│                       (RaiseEvent/Apply/Version), EventStoreRepository (stream
│                       replay + optimistic concurrency), ServiceBus,
│                       CommandSpecification test harness
├── sales/                                 BrewUp.Sales.*
│   ├── sharedkernel/{commands,events,integrationevents}   published language
│   ├── domain/{sales_order.go,commandhandlers}            event-sourced write model
│   ├── readmodel/{dtos,eventhandlers,services}            projections + queries
│   └── facade.go + module.go                              ISalesFacade + wiring
├── warehouses/                            BrewUp.Warehouses.* (same shape)
├── rest/               BrewUp.Rest — endpoints; depends ONLY on facades
├── shared/customtypes/ Quantity(100,"Lt"), Price(5,"EUR")
└── app/                composition root, graceful shutdown
```

The exact message names from the book: `CreateSalesOrder` →
`SalesOrderCreated`; `UpdateAvailabilityDueToProductionOrder` →
`AvailabilityUpdatedDueToProductionOrder`; allocation →
`BeerAvailabilityUpdated`. Domain events stay inside their module;
**integration events are separate types** even when the shape matches
(the book's Chapter 4 warning), and consumers deserialize their own
contract structs — modules never import each other.

```
Sales                          Warehouses
  │ SalesOrderCreated ─(integration)─▶ UpdateAvailabilityDueToSalesOrder per row
  │                                    │ BeerAvailabilityUpdated (remaining)
  │ ◀────────(integration)────────────┘
```

`Payment` and `Shipping` complete the book's context map and are the next
modules to add (see `docs/architecture/events.md`).

### Testing (the book's Chapter 5–7 practices)

- **Specification tests** — Given events / When command / Expect events,
  through the real handler and an in-memory event store; includes the
  book's two examples verbatim (create a sales order; 100 Lt + 100 Lt
  production → 200 Lt).
- **E2E safety net** — `Can_Create_SalesOrder`-style endpoint tests with
  eventual-consistency polling.
- **Fitness functions** — imports are parsed and asserted: Sales ⊥
  Warehouses, REST → facades only, muflone stays generic.
- **100% unit coverage, enforced** — `task cover` fails if any internal
  package drops below 100% (error branches are made reachable with fakes;
  only the `internal/app` composition root is exempt, and it is
  smoke-tested).

> **Why `internal/` and not `src/`?** In Go the layout *is* the import
> path: packages under `internal/` are un-importable from any other module
> — compiler-enforced privacy, the toolchain-native NetArchTest.

Repo scaffolding: `migrations/` (Atlas: event store + projection tables,
no cross-module FKs), `docker/` + `docker-compose.yml` (postgres → migrate
→ api → seed), `docs/` (overview, message catalog, ADRs), `.claude/` and
`.agents/` (AI-harness rules and role charters), `.env.example`.

## Where this sits in the book's journey

| Book stage | Status |
|---|---|
| Big ball of mud → modules (Ch. 6) | built modular from the start |
| Mediator between facades (Ch. 6) | skipped — superseded by the service bus, as the book itself does |
| CQRS read models (Ch. 7, branch 02) | ✅ `readmodel/` per module |
| Event sourcing + specification tests (Ch. 7, branch 03) | ✅ this repo |
| Database refactoring (Ch. 8) | schema versioned; in-memory store swappable |
| Event versioning (Ch. 11), sagas (Ch. 12), microservices (Ch. 10) | future ADRs |
