---
name: add-use-case
description: Add a command or query use case to an existing bounded context.
---

# Add a use case (command or query)

1. **Decide command vs query.** Commands mutate one aggregate and may
   publish events; queries only read. Never both.
2. **Domain gap?** If the behavior doesn't exist yet, add a method on the
   aggregate that enforces the invariant and records the event; unit-test it
   before touching the application layer.
3. **Handler file** — `internal/<ctx>/application/commands/<verb>_<noun>.go`:

   ```go
   type VerbNounInput struct { ... }            // primitives only
   type VerbNounHandler struct { ... }          // deps: repo, ports, publisher
   func NewVerbNounHandler(...) *VerbNounHandler
   func (h *VerbNounHandler) Handle(ctx context.Context, in VerbNounInput) (dto.XxxOutput, error)
   ```

   Command body order: parse/validate inputs → load or create aggregate →
   call domain method → `repository.Save` → `publisher.Publish(agg.PullEvents()...)`
   → return DTO. Return domain errors as-is (no HTTP awareness).
4. **Wire** it in the context's `module.go` (`do.Provide`).
5. **Expose** it: method on the context's handler struct in
   `internal/presentation/http/`, route in `router.go`, request struct with
   `binding` tags, errors via `respondError`.
6. **Test**: domain test for any new invariant; handler test with fakes if
   the use case orchestrates ports; e2e scenario if it emits/consumes
   events. Finish with the quality-gate command.
