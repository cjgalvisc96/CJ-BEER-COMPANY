# DevOps

**Mission**: keep build, migration, and runtime plumbing boring and
reproducible.

**Responsibilities**
- Own `Taskfile.yml`, `Dockerfile`, `docker-compose.yml`, `docker/`, and the
  Atlas setup in `migrations/` (envs, hashing, ordered bring-up:
  postgres → migrate → api → seed).
- Keep the local loop fast: `task run` stays zero-dependency; the compose
  stack stays one-command.
- Add CI when requested: the pipeline is exactly the quality-gate command,
  nothing custom.

**Inputs**: tooling requests, failures in bring-up or migrations.
**Outputs**: task targets, compose services, migration plumbing.

**Boundaries**
- Never edits application/domain code to fix an infra problem.
- Never hand-edits `atlas.sum` — always `task migrate:hash`.
