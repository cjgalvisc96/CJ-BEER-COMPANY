# Coding Style

- **Go proverbs first**: accept interfaces, return structs; small interfaces
  at the consumer side (ports live where they're needed); zero values that
  work; errors are values.
- **gofmt + go vet are the floor** — `task lint` must pass. No custom
  formatting.
- **One use case per file** in `application/commands` / `application/queries`;
  the handler struct + its input type + `Handle` method. Constructor
  injection only (no globals, no service locators outside `module.go`).
- **Value objects are immutable** — unexported fields, constructor
  validation, no setters. IDs are named types over `shared.EntityID` so the
  compiler rejects mixing a `BeerID` with an `OrderID`.
- **Money is int64 minor units + ISO currency** — never float64 for money.
- **Naming = ubiquitous language**: `Retire`, `Replenish`, `Reserve`,
  `Settle` — not `Update`, `Process`, `HandleStuff`.
- **SOLID/DRY/KISS/YAGNI as applied here**: prefer duplication over a shared
  abstraction across contexts (DRY stops at the context boundary); don't add
  configuration, generics, or indirection for futures that aren't scheduled
  (no speculative outbox, no repository base classes).
- **Comments state constraints**, not narration: why an invariant exists,
  what a race means — not what the next line does.
- **Logging**: structured `slog`, event-style keys (`orders.confirmed`),
  ids as strings. No `fmt.Println`.
