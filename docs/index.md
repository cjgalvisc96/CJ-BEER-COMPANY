# CJ Beer Company — Documentation

A craft-beer company system built as a **modular monolith** in Go with
Domain-Driven Design. Start with the [README](../README.md) for the quick
start; this directory holds the deeper material.

## Map

| Section | Contents |
|---|---|
| [architecture/overview.md](architecture/overview.md) | Bounded contexts, layering, dependency rules |
| [architecture/events.md](architecture/events.md) | Every topic on the bus, producers, consumers, payloads |
| [adr/](adr/) | Architecture Decision Records (why things are the way they are) |
| [development/testing.md](development/testing.md) | Test strategy and how to run the suites |

## The system in one paragraph

The company sells beers (**catalog**), produces them in batches (**brewing**),
keeps them in a warehouse (**inventory**), and sells them to customers
(**orders**). Contexts collaborate only through domain events on a Watermill
bus: completing a batch replenishes stock; placing an order triggers a stock
reservation whose outcome confirms or rejects the order. HTTP (Gin) is a thin
presentation layer over application-level command/query handlers wired by a
samber/do dependency-injection container.
