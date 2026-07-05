# ADR-0012: Transactional outbox, concurrency-aware retries, dead-letter redrive, trusted proxies

- **Status**: accepted
- **Date**: 2026-07-05

## Context

Three hardening gaps remained: (1) `Save` appended events and THEN
published them — a crash in between stored an event that projections
would never see (the classic dual-write); (2) optimistic-concurrency
conflicts were retried like any failure (3×) and then dead-lettered, so a
hot aggregate could poison legitimate messages; (3) dead letters were only
logged, and Gin's default proxy trust let X-Forwarded-For spoof per-IP
rate limits.

## Decisions

1. **Transactional outbox (durable mode)** — `PostgresEventStore.Append`
   writes each event's wire message into the `outbox` table IN THE SAME
   TRANSACTION as the stream append; repositories no longer publish
   directly (their publisher is nil when DB_URL is set). The
   `OutboxRelay` polls (`OUTBOX_INTERVAL`, default 250ms) with
   `FOR UPDATE SKIP LOCKED` — multi-replica safe, no sequence-gap races —
   publishes, deletes, commits. A publish failure rolls back and the rows
   are retried next tick. Delivery is now at-least-once END TO END, which
   the system's pervasive idempotency (ADR-0008/0009) absorbs by design.
   In-memory mode keeps direct publishing: a crash loses the whole store
   anyway, so there is no gap to close.
2. **Concurrency-aware retry policy** — a custom router middleware
   replaces the stock retry: generic failures get 3 attempts before the
   poison queue; `ErrConcurrency` gets 12 with exponential backoff
   (capped at 1s). Contention is an expected condition, not poison.
3. **Dead-letter archive + redrive** — in durable mode poison messages are
   persisted to `dead_letters` (topic, payload, reason) in addition to the
   log. `task redrive` (cmd/redrive) republishes un-redriven letters to
   their original topics over the shared broker and stamps them.
4. **Trusted proxies** — `TRUSTED_PROXIES` (comma-separated IPs/CIDRs)
   controls whose X-Forwarded-For is honored; the default trusts NOBODY,
   so client IPs — and the rate limiter keyed on them — cannot be spoofed.
   An invalid entry falls back to trust-none, never to trust-all.

## Consequences

- The last at-most-once window is closed; `task rebuild` demotes from
  "recovery necessity" to "operational convenience".
- Event delivery latency in durable mode is bounded by the relay interval
  (250ms per hop by default) — visible in the saga's step cadence and
  acceptable; tune `OUTBOX_INTERVAL` if needed.
- Relayed messages don't carry the original traceparent (the outbox stores
  payloads, not metadata) — spans reconnect at the consumer; a metadata
  column is the known extension if full trace continuity matters.
- Redrive requires the fault to be FIXED first; redriving still-poison
  messages just parks them again (safe loop, no loss).
