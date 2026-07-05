# quality-gate

Run the full pre-merge gate, in order, stopping at the first failure:

```bash
task tidy                # go.mod/go.sum in sync (fails CI if dirty)
task lint                # go vet + gofmt check
task check:architecture  # fitness functions: module isolation + layering
task test:race           # all tests under the race detector
task cover               # HARD GATE: 100% unit coverage (internal/app exempt)
```

Then verify the migrations are internally consistent if `migrations/` was
touched:

```bash
task migrate:hash && git diff --exit-code migrations/versions/atlas.sum
```

All green → safe to commit/PR. Any red → fix before proceeding; do not
skip with build tags or t.Skip.
