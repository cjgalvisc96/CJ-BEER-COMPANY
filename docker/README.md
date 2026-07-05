# docker/

Support files for the compose stack (`task docker:up`):

| Path | Purpose |
|---|---|
| `init/` | SQL executed once on first Postgres start (`docker-entrypoint-initdb.d`) |
| `mock-data/seed.sh` | Demo-data seeder run by the `seed` service after the API is healthy |

Bring-up order (encoded as compose `depends_on` conditions):
`postgres` (healthy) → `migrate` (Atlas, exits 0) → `api` → `seed`.
