# ADR-0001: Modular monolith in Go, one context = one vertical slice

- **Status**: accepted
- **Date**: 2026-07-05

## Context

CJ Beer Company needs catalog, brewing, inventory, and sales capabilities.
Microservices would impose network boundaries, per-service deployment, and
distributed-data problems before the domain boundaries are even proven.

## Decision

Build a single deployable Go binary structured as a modular monolith:
bounded contexts under `internal/<context>/`, each a vertical
`domain/application/infrastructure` slice with its own DI wiring
(`module.go`), mirroring the proven layout of the reference TODO app.
`internal/` gives compiler-enforced module privacy.

## Consequences

- Refactoring context boundaries is a package move, not a network migration.
- A context can be extracted later: its only inbound surface is its
  application layer + its event topics.
- Single process means the in-process GoChannel bus suffices today
  (see ADR-0002).
