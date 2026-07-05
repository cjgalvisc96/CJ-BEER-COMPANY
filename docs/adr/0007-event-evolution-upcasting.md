# ADR-0007: Event evolution via registry + upcasting (book Ch. 11)

- **Status**: accepted
- **Date**: 2026-07-05

## Context

Event-sourced history is forever: once events are the source of truth,
their shapes will outlive the code that wrote them. The book's Chapter 11
("Dealing with Events and Their Evolution") catalogs the strategies —
simple versioning, upcasting, weak schema, content negotiation,
copy-replace.

## Decision

- Every module owns a `muflone.EventRegistry` mapping stored event-type
  names to Go types; the Postgres event store rehydrates streams through
  it (nothing outside the registry can deserialize history).
- Evolution strategy: **weak schema + upcasting**. When an event's shape
  changes incompatibly, the old payload stays untouched in the store; an
  `Upcaster` registered for the old name rewrites it to the current shape
  at read time. Upcasters chain (v1 → v2 → v3), so multi-step evolutions
  compose without ever migrating stored data.
- Copy-replace (rewriting streams) is reserved for extreme cases and would
  need its own ADR.

## Consequences

- Adding a field with a sensible zero value needs nothing (weak schema —
  JSON unmarshal ignores/defaults). Renames and semantic changes need an
  upcaster plus a specification test proving old payloads still replay.
- The registry makes deserialization explicit: an unregistered event type
  fails loudly instead of decoding into the wrong struct.
- Integration events are versioned independently by topic name (they are
  already separate types per ADR-0002/0005), so a context can evolve its
  domain events without breaking consumers.
