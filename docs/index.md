# CJ Beer Company — Documentation

A craft-beer company system built as a **modular monolith** in Go with
Domain-Driven Design. Start with the [README](../README.md) for the quick
start; this directory holds the deeper material.

## Where to start

**New to these patterns (DDD, CQRS, event sourcing, sagas)?** Read
[architecture/concepts.md](architecture/concepts.md) first — it explains every
pattern in plain English with a brewery example, no prior knowledge assumed.
Then the overview and event catalog below will read easily.

## Map

| Section | Contents |
|---|---|
| [architecture/concepts.md](architecture/concepts.md) | **Start here** — every pattern in plain English, and where it lives |
| [architecture/overview.md](architecture/overview.md) | Bounded contexts, module layout, layering, dependency rules |
| [architecture/events.md](architecture/events.md) | Every topic on the bus, producers, consumers, payloads |
| [adr/](adr/) | Architecture Decision Records (why things are the way they are) |
| [development/testing.md](development/testing.md) | Test strategy and how to run the suites |

## The system in one paragraph

The Go rendition of **BrewUp**, the brewery application from
*Domain-Driven Refactoring*, at the book's end state: a modular monolith
with **CQRS and event sourcing**. The **Sales** module takes orders
(`CreateSalesOrder` → `SalesOrderCreated`); the **Warehouses** module
tracks beer availability (production orders fill it, sales orders allocate
from it via integration events). Aggregates are event-sourced through a
Muflone-style framework (`internal/muflone`), reads are eventually
consistent projections, Gin maps the `/v1/sales` and `/v1/warehouses`
endpoints through module facades, and samber/do wires it together.
