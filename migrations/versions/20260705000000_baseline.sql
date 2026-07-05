-- Baseline schema for CJ Beer Company (BrewUp-style CQRS + Event Sourcing).
--
-- The write model is an EVENT STORE: append-only streams of domain events,
-- one stream per aggregate (SalesOrder-<id>, Availability-<id>), with
-- optimistic concurrency on (stream_id, version). It is the source of
-- truth; the read-model tables are projections that can be rebuilt from it
-- at any time.
--
-- As in the in-memory layout, each module owns its data: no foreign keys
-- cross module boundaries, so a module can move to its own database
-- without DDL surgery.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- write model: event store --------------------------------------------------
CREATE TABLE "events" (
    "stream_id"   varchar(100) NOT NULL,
    "version"     integer      NOT NULL CHECK ("version" > 0),
    "commit_id"   uuid         NOT NULL,
    "event_type"  varchar(120) NOT NULL,
    "payload"     jsonb        NOT NULL,
    "occurred_at" timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY ("stream_id", "version")
);
CREATE INDEX "ix_events_commit_id" ON "events" ("commit_id");
CREATE INDEX "ix_events_event_type" ON "events" ("event_type");

-- read model: sales ----------------------------------------------------------
CREATE TABLE "sales_orders" (
    "id"                 uuid PRIMARY KEY,
    "sales_order_number" varchar(50)  NOT NULL,
    "order_date"         timestamptz  NOT NULL,
    "customer_id"        uuid         NOT NULL,
    "customer_name"      varchar(200) NOT NULL,
    "projected_at"       timestamptz  NOT NULL DEFAULT now()
);

CREATE TABLE "sales_order_rows" (
    "sales_order_id"  uuid         NOT NULL REFERENCES "sales_orders" ("id") ON DELETE CASCADE,
    "beer_id"         uuid         NOT NULL,
    "beer_name"       varchar(200) NOT NULL,
    "quantity"        integer      NOT NULL,
    "unit_of_measure" varchar(10)  NOT NULL,
    "price"           numeric(12,2) NOT NULL,
    "currency"        char(3)      NOT NULL,
    PRIMARY KEY ("sales_order_id", "beer_id")
);

-- read model: warehouses -----------------------------------------------------
CREATE TABLE "availabilities" (
    "beer_id"         uuid PRIMARY KEY,
    "beer_name"       varchar(200) NOT NULL,
    "quantity"        integer      NOT NULL,
    "unit_of_measure" varchar(10)  NOT NULL,
    "projected_at"    timestamptz  NOT NULL DEFAULT now()
);
