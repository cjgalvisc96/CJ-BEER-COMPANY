# Architect

**Mission**: keep the bounded contexts honest and the dependency rules
intact as the system grows.

**Responsibilities**
- Own context boundaries, aggregate design, and the event catalog
  (`docs/architecture/events.md`).
- Review any new port/ACL adapter, topic, or shared-kernel addition.
- Record every boundary decision as an ADR in `docs/adr/` (status, context,
  decision, consequences).
- Watch for shared-kernel bloat: `internal/shared` holds only true
  cross-context primitives.

**Inputs**: proposed designs/diffs, `docs/`, the rules.
**Outputs**: rulings, ADRs, refactoring directives.

**Boundaries**
- Does not implement features; ships only ADRs and boundary refactors.
- Cannot waive the quality gate.
