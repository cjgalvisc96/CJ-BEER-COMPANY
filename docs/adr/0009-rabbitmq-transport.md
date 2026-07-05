# ADR-0009: Pluggable service-bus transport; RabbitMQ in production

- **Status**: accepted (closes the "transport property" caveat of ADR-0008)
- **Date**: 2026-07-05

## Context

With the in-process GoChannel bus, messages in flight died with the
process — the event store plus the saga resumer compensated, but delivery
itself was not durable. The book's BrewUp uses RabbitMQ
(Muflone.Transport.RabbitMQ) as its service bus.

## Decision

- A `Transport` port inside muflone (`Publisher()`, `SubscriberFor(handler)`,
  `Close()`) decouples the ServiceBus from the wire. Handler subscriptions
  attach lazily at `Run`, when a broker must be reachable.
- **`InMemoryTransport`** (GoChannel) stays the default: dev and tests run
  with zero dependencies.
- **`AMQPTransport`** (watermill-amqp) is the production wire, selected by
  `BROKER_URL`: each topic is a fanout exchange; each handler binds its
  own **durable queue** (`<topic>_<handler>`). That yields fan-out per
  handler on shared topics and **competing consumers** across replicas of
  the same handler — consumer-group semantics, ready for horizontal
  scaling. The dead-letter topic (ADR-0008) becomes a durable queue too.
- Kafka (or NATS) is another `Transport` implementation when needed — the
  bus, handlers, and modules are untouched by the wire.

## Consequences

- Messages now survive restarts alongside the event store: the compose
  stack runs Postgres + RabbitMQ, and the full saga flow (including
  compensation) is verified over the real broker in the smoke test.
- The transport glue is unit-tested through constructor seams (fakes
  proving config, queue naming, error paths, shutdown); the real wire is
  integration-tested by compose/CI — the honest split for code whose
  behavior lives in a broker.
- At-least-once delivery is now real across processes: the idempotency
  guarantees of ADR-0008 (saga records, availability holds, settlement)
  stop being defensive and become load-bearing.
- The app fails fast at boot if `BROKER_URL` is set but unreachable, same
  policy as the database.
