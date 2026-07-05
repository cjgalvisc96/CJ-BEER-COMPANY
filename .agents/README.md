# Agent Harness

Work on this repo is organized as a small team of role-specialized agents
coordinated by a **Lead**. Each role's charter (`role.md`) defines its
mission, inputs/outputs, and — most importantly — its boundaries.

| Role | File | Owns |
|------|------|------|
| **Lead** | [`lead/role.md`](lead/role.md) | Coordination, sequencing, final decisions |
| **Developer** | [`developer/role.md`](developer/role.md) | Implementation within layering/context rules |
| **Architect** | [`architect/role.md`](architect/role.md) | DDD boundaries, ADRs, event contracts |
| **Tester** | [`tester/role.md`](tester/role.md) | Test pyramid, race safety, e2e choreography |
| **DevOps** | [`devops/role.md`](devops/role.md) | Taskfile, Docker/compose, Atlas migrations, CI |
| **SRE** | [`sre/role.md`](sre/role.md) | Logging, health, graceful shutdown, reliability |

## How to use the roles

Give an agent its `role.md` plus the `.claude/rules/*` files as context.
The Lead decomposes the request, routes work to roles, and merges results;
any cross-boundary disagreement escalates to the Architect, whose ruling is
recorded as an ADR in `docs/adr/`.
