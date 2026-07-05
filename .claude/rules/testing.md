# Testing Rules

- **Every aggregate rule gets a domain unit test** — including recorded
  events (`PullEvents` content) and idempotency/no-op paths.
- **Use cases test against fakes, not mocks** — hand-rolled fakes of the
  ports (see `place_order_test.go`); if a fake exceeds ~20 lines, the port
  is too big.
- **Cross-context behavior is proven end-to-end** — extend
  `tests/e2e_flow_test.go` through the HTTP API; assert async outcomes with
  `require.Eventually`, never `time.Sleep`.
- **`task test:race` must pass** — new repository adapters return snapshots
  (records → rehydrated entities), never shared aggregate pointers.
- Tests are black-box (`package x_test`), table-driven where it helps, and
  use `testify` (`require` for preconditions, `assert` for verdicts).
- Don't test getters, DTO mapping, or the DI wiring in isolation — the e2e
  suite covers wiring by construction.
