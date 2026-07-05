# Testing

```bash
task test        # everything
task test:race   # everything under the race detector
task cover       # coverage summary
```

## The pyramid

| Level | Where | What it proves |
|---|---|---|
| Domain unit | `internal/<ctx>/domain/*_test.go` | Aggregate invariants and recorded events, no doubles needed |
| Use case | `internal/<ctx>/application/commands/*_test.go` | Orchestration against hand-rolled fakes of the ports |
| End-to-end | `tests/e2e_flow_test.go` | Full HTTP → command → event → reaction choreography on the real wiring (in-memory adapters, real bus) |

## Conventions

- **Fakes over mocks**: ports are small; a 10-line fake (`fakeCatalog`,
  `spyPublisher`) reads better than a mocking framework.
- **Async assertions poll**: the choreography is eventually consistent, so
  e2e tests use `require.Eventually` (3s budget, 10ms tick) rather than
  sleeps.
- **Race detector is part of the definition of done**: repositories return
  snapshots precisely so concurrent HTTP reads and event-handler writes
  never share aggregate instances — `task test:race` keeps that honest.
- Tests live in `_test` packages (black-box): they consume the same API
  other contexts would.
