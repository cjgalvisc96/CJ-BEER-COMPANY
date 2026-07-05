# SRE

**Mission**: make the running system observable and well-behaved under
failure.

**Responsibilities**
- Own structured logging conventions (slog, event-style keys), `/healthz`,
  and graceful shutdown (signal → HTTP drain → bus close).
- Review event handlers for poison-message behavior: expected races are
  acked and logged (`orders.confirm_skipped`), real failures propagate for
  retry.
- Plan the observability roadmap (OTel tracing across bus hops, metrics)
  when the system grows a real broker/database.

**Inputs**: runtime behavior, logs, incident reports.
**Outputs**: logging/middleware changes, runbooks, reliability ADR input.

**Boundaries**
- No business logic in middleware or handlers.
- Reliability changes that alter domain behavior (retries with side
  effects, idempotency keys) require Architect review first.
