# The patterns, in plain English

This project uses a handful of architecture patterns that sound intimidating
but are simple ideas once you see them. This page explains each one in plain
language, with a brewery example and a pointer to where it lives in the code.
No prior Domain-Driven Design knowledge needed — read this first, then the
[overview](overview.md) and [message catalog](events.md) will make sense.

The running example: **a brewery produces beer, customers order it, and the
warehouse hands stock out until it runs low.** Everything below is in service
of doing that correctly, even when things fail.

---

## The shape of the app

**Modular monolith.** One program you deploy, split inside into independent
*modules* that can't reach into each other's code. You get the simplicity of a
single app to run, with the discipline of microservices' boundaries — and if a
module ever needs to become its own service, the seam is already there.
*Here:* `internal/sales` and `internal/warehouses`. A test literally fails the
build if one imports the other (`tests/architecture_test.go`).

**Bounded context (module).** A part of the business with its own language and
rules. "Order" means something different to Sales (a customer's purchase) than
to the Warehouse (stock to hand out), so they're kept separate and talk only by
sending messages.
*Here:* the two modules above; `Payment` and `Shipping` exist in the book's map
but aren't built yet.

**Ubiquitous language.** Code uses the exact words the business uses —
`Reserve`, `Replenish`, `Settle`, `Retire` — never vague tech verbs like
`Update` or `Process`. Reading the code should sound like talking to a brewer.

**Facade.** The one public "front desk" of a module. Everything outside (like
the web layer) goes through it and never touches the module's insides, so the
insides stay free to change.
*Here:* `sales.Facade`, `warehouses.Facade`. The REST layer is only allowed to
call these (again, enforced by a test).

---

## The write side: how things change

**Aggregate.** A single consistency boundary — one object that guards a set of
rules and is always saved and loaded as a whole. You never change part of it
from outside; you ask it to do something and it decides.
*Here:* `SalesOrder` and `Availability`. `Availability` is what refuses to
allocate beer that isn't in stock.

**Command vs event.**
- A **command** is a *request* to do something — imperative, future-tense,
  might be refused: `CreateSalesOrder`. Exactly one handler takes it.
- An **event** is a *fact* that already happened — past-tense, undeniable:
  `SalesOrderCreated`. Anyone interested can react to it.

**Event sourcing.** Instead of storing the *current state* ("stock = 70 Lt"),
we store the *list of things that happened* ("+100 produced, −30 allocated")
and add them up to get the state. Nothing is ever overwritten, so you have a
perfect history and can rebuild any view from scratch.
*Here:* the aggregate changes state **only** by raising an event and applying
it — there are no setters. `RaiseEvent(...)` + `apply(...)` in the domain
files. The list of events for one order lives in a *stream* called
`SalesOrder-<id>`.

**Repository.** The thing that loads an aggregate (replay its event stream) and
saves it (append the new events). The write side runs *no queries* — it only
ever loads one aggregate by id.
*Here:* `muflone.EventStoreRepository`. Two backends: in-memory for dev/tests,
Postgres for production (chosen by the `DB_URL` setting).

**Optimistic concurrency + retries.** If two requests try to change the same
aggregate at once, the second one to save is rejected (its version is stale)
rather than silently clobbering the first. The bus simply retries it — a normal,
expected hiccup, not an error. Genuine contention gets *more* patience (12
retries) than a random failure (3) before giving up.
*Here:* `ErrConcurrency` in the event store; the retry policy in
`muflone/retry.go`.

---

## The read side: how you query

**CQRS** (Command Query Responsibility Segregation). A fancy name for one rule:
**the code that changes data and the code that reads data are separate.** Writes
go through aggregates and events (above); reads come from purpose-built tables.
Neither gets in the other's way.

**Read model / projection.** A plain, query-friendly copy of the data, built by
*listening to events* and writing them into a simple table. It exists only to be
read from — it has no business rules.
*Here:* when `SalesOrderCreated` happens, a projection writes a row into the
sales-order read model. `GET /v1/sales` reads that table, never the event
streams.

**Eventual consistency.** Because the read side is updated by reacting to
events, it lags the write side by a heartbeat. So a write returns the new id
immediately, and you poll the read model until your change shows up — instead of
waiting for everything to finish in one request.
*Here:* `POST /v1/sales` returns the order id right away; tests use
`require.Eventually` to wait for the projection.

**Rebuild.** Because the events are the source of truth, any read model can be
thrown away and reconstructed by replaying history. Useful when you add a new
view or fix a projection bug.
*Here:* `task rebuild`.

---

## Talking between modules

**Service bus.** The in-app "post office" that carries messages between parts.
Two delivery styles: **producer-consumer** for commands (one recipient) and
**publish-subscribe** for events (everyone interested).
*Here:* `muflone.ServiceBus`. It runs in-process by default; set `BROKER_URL`
and it rides **RabbitMQ** instead, so messages survive restarts and multiple
copies of the app can share the workload.

**Domain event vs integration event.** A *domain* event is private to the module
that raised it. When another module needs to know, you publish a *separate*
**integration** event — a deliberate, stable, public announcement — even if it
looks identical. This stops one module's internal details from leaking into
another's and becoming an accidental contract.
*Here:* Sales raises the domain event `SalesOrderCreated`, then re-publishes it
as an *integration* event that Warehouses subscribes to.

**Choreography.** No central conductor tells modules what to do; each reacts to
events on its own. Sales doesn't call Warehouses — it announces "order created"
and Warehouses decides to act.

---

## When one action spans modules: the saga

**Saga (process manager) + compensating transactions.** Some goals need several
steps across modules, and you can't wrap them in one database transaction. A
**saga** is a little state machine that walks the steps, and if a later step
fails, it runs **compensations** — explicit "undo" actions — for the steps that
already succeeded. (You can't roll back across services, so you *apologize*
instead of pretending it never happened.)
*Here:* `OrderAllocationSaga`. To fill an order it allocates stock row by row.
If one row is short (`QuantityNotFound`), it *gives back* the rows it already
took and rejects the order. The saga itself is event-sourced, so its progress is
durable. (Book Chapter 12; ADR-0008.)

**Durable execution.** The saga must survive crashes. So: in-flight sagas
**resume** when the app boots, a **watchdog** re-nudges steps that have stalled
too long (`SAGA_STEP_TIMEOUT`), and a message that keeps failing is parked on a
**dead-letter** queue instead of blocking the line or being lost.
*Here:* `ResumeInFlight` / `TimeoutInFlight` in the saga handler.

---

## Not losing messages

**Idempotency.** Handling the same message twice has the same effect as handling
it once. This is what makes "retry on failure" safe — a redelivered
`SalesOrderCreated` won't create a second order.
*Here:* command handlers check "does this aggregate already exist?"; the
warehouse remembers which orders it has already served.

**Transactional outbox.** A subtle bug: if you save an event to the database and
*then* publish it to the bus, a crash in between loses the message forever (or a
crash the other way sends a message for a change that got rolled back). The fix:
in the **same database transaction** that saves the event, write a row into an
`outbox` table. A separate **relay** reads the outbox and publishes — so saving
and "promising to publish" happen atomically, and nothing can fall through the
gap.
*Here:* durable mode writes to `outbox` inside the append transaction;
`OutboxRelay` publishes and clears it (`OUTBOX_INTERVAL`). ADR-0012.

**Dead letters + redrive.** A message that fails past its retries is archived
(not dropped) with the reason. Once you've fixed the cause, you replay the
archive back onto the bus.
*Here:* the `dead_letters` table; `task redrive`.

---

## Evolving safely

**Event upcasting.** Events are stored forever, so old events must still load
after you change an event's shape. An **upcaster** is a small function that
transforms an old stored version into the current one as it's read — so today's
code understands yesterday's events without touching the database.
*Here:* `muflone.RegisterUpcaster`. Book Chapter 11; ADR-0007.

---

## Running it in production

**Two modes, one binary.** The same build runs zero-dependency for dev/tests and
fully durable in production, chosen entirely by environment variables:

| Setting | Empty (dev/tests) | Set (production) |
|---|---|---|
| `DB_URL` | everything in memory | Postgres event store + read models |
| `BROKER_URL` | in-process bus | RabbitMQ (durable, multi-replica) |
| `AUTH_ISSUER` | open API | OIDC tokens + RBAC |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | metrics only | metrics + exported traces |

**Authentication + RBAC.** In production every `/v1` route needs a signed bearer
token (OIDC, from Keycloak), and *roles* decide who can do what: a `viewer`
reads, a `sales-manager` places orders, a `warehouse-operator` declares
production. ADR-0010.

**Observability.** You can see what the running app is doing three ways, all
tagged with `APP_ENV` (`local`/`dev`/`staging`/`prod`) so environments never get
confused:
- **Logs** — structured JSON, every line carrying the environment.
- **Metrics** — Prometheus at `/metrics`.
- **Traces** — the path of one request across HTTP and bus hops (when an OTLP
  endpoint is set), grouped by `deployment.environment`.

**API hardening.** The edge is defended: **idempotent** order creation (safe to
retry), **pagination** on lists, per-IP **rate limiting** and request **body
caps**, and **spoof-proof client IPs** (a forwarded IP is trusted only from a
proxy you explicitly list in `TRUSTED_PROXIES`). ADR-0011.

---

## Guardrails that keep it honest

**Fitness functions.** Automated tests that assert the *architecture itself*,
not behavior — "Sales must not import Warehouses," "REST may only call facades."
The rules can't rot because the build fails the moment they're broken.
*Here:* `tests/architecture_test.go` (`task check:architecture`).

**Specification tests.** The way aggregate behavior is proven: **Given** these
past events, **When** this command arrives, **Expect** exactly these new events
(or this refusal). Reads like a spec, runs through the real handler.
*Here:* the `*_specification_test.go` files; harness in `muflone/specification.go`.

**100% unit coverage, enforced.** `task cover` fails the build unless every
function is fully covered (two small exemptions: `cmd/` and the composition
root). It's a gate in CI, not a suggestion.

---

Ready for the details? → [architecture overview](overview.md) ·
[message catalog](events.md) · [why each decision was made](../adr/) ·
[testing](../development/testing.md)
