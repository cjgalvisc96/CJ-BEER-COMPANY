---
name: add-bounded-context
description: Add a new BrewUp-style module (bounded context) to the CJ-BEER-COMPANY modular monolith — e.g. Payment or Shipping from the book's context map.
---

# Add a module (bounded context)

Mirror an existing module (`internal/warehouses` is the complete example).

1. **Shared kernel** — `internal/<module>/sharedkernel/`:
   - `customtypes.go`: strongly named ids/values (`PaymentId{Value uuid.UUID}`).
   - `commands/`: imperative structs embedding `muflone.CommandBase`
     (aggregateId + commitId) with a `MessageName()` like
     `"<module>.<verb_noun>"` and a `New*` constructor.
   - `events/`: past-tense structs embedding `muflone.DomainEventBase`.
   - `integrationevents/`: separate types for anything shared outward.
2. **Domain** — `internal/<module>/domain/`:
   - Event-sourced aggregate: `StreamName` const, `New<Aggregate>()`
     binding the router, factory validating invariants + `RaiseEvent`,
     `Route` type-switch dispatching to apply methods (the ONLY place
     state is assigned).
   - `commandhandlers/`: one handler per command — GetByID (or create on
     `muflone.ErrAggregateNotFound`), call the aggregate method,
     `repository.Save(ctx, aggregate, uuid.New())`. Business refusals log
     + return nil; infrastructure errors propagate.
3. **Read model** — `internal/<module>/readmodel/`:
   - `dtos/` (query shapes), `services/` (projection writer + queries,
     `sync.RWMutex`), `eventhandlers/` (project each domain event).
4. **Facade + module** — `facade.go` (inbound JSON shape, command
   building, query pass-through) and `module.go` (`Register(injector, bus)`:
   event store, repository, `muflone.RegisterCommandHandler`,
   `RegisterDomainEventHandler` for projections/integration publishers,
   `SubscribeIntegrationEvent` for inbound reactions, provide the Facade).
5. **Wire** in `internal/app/app.go`; map endpoints in `internal/rest`
   (facade only!).
6. **Persist** — projection tables in a new Atlas migration (no FKs to
   other modules), then `task migrate:hash`.
7. **Test** — specification tests per command (Given/When/Expect), an e2e
   scenario if it joins the choreography, and extend
   `tests/architecture_test.go` with the new module's isolation rules.
8. Update `docs/architecture/overview.md` + `events.md`; run quality-gate.
