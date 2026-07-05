---
name: add-use-case
description: Add a command (write) or query (read) to an existing module, BrewUp-style.
---

# Add a use case

## Command (write side)

1. **Command** in `sharedkernel/commands/`: struct embedding
   `muflone.CommandBase`, imperative `MessageName()`
   (`"<module>.<verb_noun>"`), `New*` constructor taking the typed
   aggregate id + `commitId uuid.UUID`.
2. **Domain event(s)** in `sharedkernel/events/`: past tense, embedding
   `muflone.DomainEventBase`; carry the resulting state (the book's
   availability events carry the NEW cumulative quantity).
3. **Aggregate method** in `domain/`: validate the invariant, then
   `RaiseEvent(...)`. Add the event to the `Route` type-switch and write
   its apply method — state is assigned only there.
4. **Command handler** in `domain/commandhandlers/`: load (or create) the
   aggregate, call the method, `Save(ctx, aggregate, uuid.New())`.
   Register it in `module.go` with `muflone.RegisterCommandHandler`.
5. **Projection**: extend/add a read-model event handler + dto/service,
   register with `muflone.RegisterDomainEventHandler`.
6. **Integration**: if other contexts must react, add a SEPARATE type in
   `sharedkernel/integrationevents/` and a publisher handler in
   `module.go` — never publish the domain event outward.
7. **Specification test** (Given/When/OnHandler/Expect) — including the
   refusal path (`ExpectedError` or ack-with-nothing-committed).
8. Facade + endpoint if user-facing; e2e test if it joins a flow.

## Query (read side)

Add the method to the module's read-model service + facade, expose it in
`internal/rest`. Queries never touch the event store or the aggregate —
read models only.
