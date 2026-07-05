-- Runs once on first postgres container start (docker-entrypoint-initdb.d).
-- pgcrypto is also created by the baseline migration; having it here keeps
-- ad-hoc psql sessions working before migrations run.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
