# Architecture Rules (non-negotiable)

This repo is the Go rendition of the book's BrewUp application (modular
monolith + CQRS + event sourcing). Keep it that way.

1. **Module isolation** — `internal/sales` and `internal/warehouses` never
   import each other (fitness test enforced). Cross-module communication
   is integration events on the service bus; consumers own their contract
   structs.
2. **Module shape** — every module keeps the BrewUp split:
   `sharedkernel/` (customtypes, commands, events, integrationevents),
   `domain/` (event-sourced aggregate + commandhandlers), `readmodel/`
   (dtos, eventhandlers, services), `facade.go`, `module.go`.
3. **Event sourcing is the only write path** — aggregates change state
   exclusively via `RaiseEvent` + apply methods; no setters, no direct
   field writes outside apply. Repositories are `muflone.Repository`
   (GetByID/Save) — the write model runs no queries.
4. **Commands vs events** — commands imperative (`CreateSalesOrder`), one
   handler, producer-consumer; events past tense (`SalesOrderCreated`),
   pub/sub. Both carry `aggregateId` + `commitId`.
5. **Domain events stay home; integration events are separate types** even
   when the shape matches — never publish a domain event across contexts.
6. **REST depends only on facades** (fitness test enforced). No business
   logic in endpoints; facades build commands and query read models.
7. **muflone stays generic** — it must never import a business module.
8. **Reads are eventually consistent** — writes return the pre-generated
   aggregate id; callers poll projections. Never make a command handler
   feed a response synchronously.
9. **Migrations** (Atlas, `migrations/`): event store + projection tables,
   no FKs across modules; new versioned migration + `task migrate:hash`.
