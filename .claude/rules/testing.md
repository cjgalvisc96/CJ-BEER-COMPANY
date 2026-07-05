# Testing Rules

- **Every aggregate behavior gets a specification test** —
  `muflone.CommandSpecification`: Given past events, When a command,
  Expect the committed events (or `ExpectedError` + nothing committed).
  Run through the real command handler; never mock the aggregate.
- **Business refusals ack, failures nack** — a domain refusal (e.g. not
  enough stock) logs and commits nothing but returns nil to the bus; only
  infrastructure failures propagate for redelivery. Test both.
- **Cross-module behavior is proven end-to-end** in `tests/` through the
  HTTP endpoints, asserting on read models with `require.Eventually` —
  never bare sleeps as assertions.
- **Fitness functions guard the structure** (`tests/architecture_test.go`);
  extend them when adding a module.
- **`task test:race` must pass** — read-model services and the event store
  are shared by HTTP readers and bus handlers.
- Tests are black-box (`package x_test`), testify style (`require` for
  preconditions, `assert` for verdicts).
