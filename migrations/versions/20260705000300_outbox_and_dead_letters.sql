-- Transactional outbox (ADR-0012): the wire message commits atomically
-- with the event-stream append; the relay publishes and deletes. Dead
-- letters are archived durably for inspection and `task redrive`.
CREATE TABLE "outbox" (
    "id"         bigserial PRIMARY KEY,
    "topic"      varchar(200) NOT NULL,
    "payload"    jsonb        NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now()
);

CREATE TABLE "dead_letters" (
    "id"          bigserial PRIMARY KEY,
    "topic"       varchar(200) NOT NULL,
    "payload"     jsonb        NOT NULL,
    "reason"      text         NOT NULL DEFAULT '',
    "created_at"  timestamptz  NOT NULL DEFAULT now(),
    "redriven_at" timestamptz
);
CREATE INDEX "ix_dead_letters_pending" ON "dead_letters" ("redriven_at") WHERE "redriven_at" IS NULL;
