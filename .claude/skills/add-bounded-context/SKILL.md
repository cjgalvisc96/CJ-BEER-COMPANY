---
name: add-bounded-context
description: Add a new bounded context to the CJ-BEER-COMPANY modular monolith following the established vertical-slice layout.
---

# Add a bounded context

Follow the shape of an existing context (`internal/inventory` is the
smallest complete example).

1. **Domain first** — `internal/<ctx>/domain/`:
   - Aggregate root embedding `shared.AggregateRoot`; unexported fields;
     behavior methods that validate and `RecordEvent`.
   - Typed ID over `shared.EntityID`; opaque `XxxRef` types for other
     contexts' aggregates.
   - `events.go` (topic consts + event structs), `errors.go` (shared error
     kinds), `repository.go` (the port), `Rehydrate*` constructor for
     persistence.
2. **Application** — `application/commands/*.go` (one handler per file:
   parse → load/build aggregate → mutate → `Save` → `Publish(PullEvents())`),
   `application/queries/`, `application/dto/`.
   - Needs data from another context? Define a port in `application/ports`
     with a context-local snapshot struct.
   - Reacts to another context's events? Add `application/eventhandlers`
     with local contract structs of only the fields consumed.
3. **Infrastructure** — `infrastructure/persistence/memory_<agg>_repository.go`
   (record snapshot + `sync.RWMutex`, rehydrate on read);
   `infrastructure/acl/` adapters for the ports (import the other context's
   *application queries* only).
4. **Wire** — `module.go` with `Register(do.Injector)` (+
   `SubscribeEventHandlers(injector, bus)` if it consumes events). Call both
   from `internal/app/app.go`.
5. **Expose** — handlers in `internal/presentation/http/<agg>_handlers.go`,
   routes in `router.go` under `/api/v1/...`.
6. **Persist** — new versioned migration in `migrations/versions` (no FKs to
   other contexts' tables), then `task migrate:hash`.
7. **Test** — domain unit tests, a use-case test with fakes if it has ports,
   and an e2e scenario in `tests/` if it participates in a choreography.
8. Update `docs/architecture/overview.md` + `events.md`; run the
   quality-gate command.
