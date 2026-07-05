# Architecture Rules (non-negotiable)

1. **Layer direction** — inside a context: `domain` imports only
   `internal/shared/domain`; `application` imports its own domain (+ shared
   ports); `infrastructure` implements ports. Nothing imports
   `infrastructure` except the context's `module.go`.
2. **Context isolation** — a context may import another context **only**
   from `infrastructure/acl`, and only the other context's `application`
   layer. Domain types never cross context boundaries; use opaque
   `BeerRef`-style IDs.
3. **Events are the public contract** — cross-context reactions subscribe to
   topic strings and unmarshal local consumer-driven structs. Never import a
   producer's event types into a consumer.
4. **Events after persistence** — command handlers publish
   `aggregate.PullEvents()` only after a successful `repository.Save`.
5. **Behavior on aggregates** — business rules live in domain methods
   (`Order.Confirm()`, `StockItem.Reserve()`), not in services or handlers.
   No anemic models. Constructors/`Parse*` reject invalid values; there is
   no way to hold an invalid value object.
6. **Presentation is thin** — Gin handlers bind → call one command/query
   handler → map errors via `respondError`. No business logic, no domain
   imports.
7. **Errors** — construct via `shared` error kinds
   (`NewValidationError`, `NewNotFoundError`, `NewConflictError`,
   `NewUnprocessableError`); wrap with `fmt.Errorf("%w: context")`. HTTP
   mapping happens only in `presentation/http/respond.go`.
8. **Schema mirrors the model** — migrations (Atlas, `migrations/`) carry no
   cross-context foreign keys (ADR-0004). New tables for a context go in a
   new versioned migration; run `task migrate:hash` after edits.
9. **Everything under `internal/`** — never create importable public
   packages without an explicit decision (ADR).
