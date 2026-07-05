# ADR-0011: API hardening and operability

- **Status**: accepted
- **Date**: 2026-07-05

## Context

The system was architecturally complete (books Ch. 1–12 + auth) but had
operational gaps for a real front door: unsafe client retries, unbounded
list responses, no protection against abusive requests, no metrics or
distributed traces, and no way to exercise event sourcing's signature
capability — rebuilding read models.

## Decisions

1. **Idempotent order creation** — `POST /v1/sales` accepts an optional
   client-supplied `id` (the aggregateId); the command handler
   acknowledges an already-existing order instead of duplicating or
   erroring. Retries after network timeouts are safe end to end.
2. **Pagination** — list endpoints take `?limit=&offset=` (clamped by the
   `Page` value object: default 50, max 200) and answer with an
   `{items, limit, offset}` envelope. Both read-model adapters paginate
   (slice window in memory, LIMIT/OFFSET in Postgres).
3. **Projection rebuild** — `task rebuild` (cmd/rebuild) resets the
   projection tables and replays every stream through the same projection
   handlers the bus uses. Each module exposes `NewEventRegistry()` and
   `RebuildReadModel(...)`; saga streams are skipped (the saga IS its
   stream).
4. **Observability** — Prometheus metrics always on at `/metrics`
   (request count/duration via the OTel meter); OTLP trace export opt-in
   via `OTEL_EXPORTER_OTLP_ENDPOINT`, with W3C trace context propagated
   **through the service bus** (publish/consume spans, `traceparent` in
   message metadata) so a trace follows command → event → saga → event.
5. **Request guards** — per-client-IP token-bucket rate limiting
   (`RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`, 429) and a request body cap
   (`MAX_BODY_BYTES`, 413). Single-node guards by design; a gateway/LB
   owns the global limits in a bigger topology.

## Consequences

- Every knob follows the house rule: zero value = disabled/default, env
  var = production behavior; dev and the in-process test suite stay
  zero-dependency.
- The REST constructor takes an `Options` struct — cross-cutting concerns
  stopped accreting positional parameters.
- List consumers must read `.items` (breaking change to the raw-array
  responses; the seeder, e2e suite, and CI smoke were updated).
- The rebuild command is the recovery runbook for corrupted read models
  and the migration path for brand-new projections.
