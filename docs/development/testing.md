# Testing

```bash
task test        # everything
task test:race   # everything under the race detector
task cover       # coverage summary
```

## The pyramid (as the book builds it)

| Level | Where | What it proves |
|---|---|---|
| **Specification tests** (book Ch. 7) | `internal/<module>/domain/commandhandlers/*_specification_test.go` | The whole aggregate lifecycle: **Given** past events, **When** a command, **Expect** committed events — run through the real command handler and an in-memory event store, no mocks |
| **E2E tests** (book Ch. 5–6, the safety net) | `tests/e2e_flow_test.go` | Endpoints behave like production: POST → Created, projections appear, the cross-module flow settles |
| **Fitness functions** (book Ch. 6) | `tests/architecture_test.go` (`task check:architecture`) | Module isolation (Sales ⊥ Warehouses), REST-only-through-facades, muflone stays generic, shared custom types stay leaf, and the per-module SharedKernel → Domain → ReadModel layering all survive refactoring |

## Coverage gate — 100%, enforced

`task cover` (scripts/coverage_gate.sh, also run in CI) collects coverage
across all internal packages with `-coverpkg` and **fails unless every
function sits at 100%**. Two exemptions only: `cmd/` (the main shell) and
`internal/app` (composition-root wiring whose remaining branches are
runtime fault-injection paths — it is smoke-tested in
`internal/app/app_test.go`). When you add an error branch, make it
reachable with a fake (failing store, closed bus, failing projection
writer) instead of leaving it uncovered.

## Conventions

- Specification tests mirror Muflone's `CommandSpecification<TCommand>`:
  implement `Given`/`When`/`OnHandler`/`Expect` and call `.Run(t)`.
  Events are compared by type and value (the book's `CompareEvents`).
- Async outcomes poll with `require.Eventually` — never `time.Sleep` as an
  assertion (one deliberate settle-sleep exists for a negative case).
- `task test:race` is part of the definition of done.
- Tests live in `_test` packages (black-box).
