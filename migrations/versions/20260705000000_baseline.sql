-- Baseline schema for CJ Beer Company: one table set per bounded context.
--
-- Deliberate design choices (mirroring the modular-monolith boundaries):
--   * beer_id columns in inventory/brewing/orders are OPAQUE references to
--     the catalog context — no cross-context foreign keys, so each context
--     can later move to its own schema or database without breaking DDL.
--   * The only FK is order_lines → orders: they live inside the same
--     aggregate boundary.
--   * Money is stored as minor units (cents) + ISO currency, matching the
--     shared.Money value object.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- catalog ---------------------------------------------------------------------
CREATE TABLE "beers" (
    "id"          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "name"        varchar(200) NOT NULL,
    "style"       varchar(20)  NOT NULL,
    "abv"         numeric(4,1) NOT NULL CHECK ("abv" >= 0 AND "abv" <= 20),
    "price_cents" bigint       NOT NULL CHECK ("price_cents" >= 0),
    "currency"    char(3)      NOT NULL,
    "description" text         NOT NULL DEFAULT '',
    "status"      varchar(10)  NOT NULL DEFAULT 'active',
    "created_at"  timestamptz  NOT NULL DEFAULT now(),
    "updated_at"  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT "uq_beers_name" UNIQUE ("name")
);
CREATE INDEX "ix_beers_status" ON "beers" ("status");

-- inventory -------------------------------------------------------------------
CREATE TABLE "stock_items" (
    "beer_id"       uuid PRIMARY KEY,
    "quantity"      integer     NOT NULL DEFAULT 0 CHECK ("quantity" >= 0),
    "reorder_level" integer     NOT NULL DEFAULT 0 CHECK ("reorder_level" >= 0),
    "updated_at"    timestamptz NOT NULL DEFAULT now()
);

-- brewing ---------------------------------------------------------------------
CREATE TABLE "batches" (
    "id"           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "beer_id"      uuid        NOT NULL,
    "units"        integer     NOT NULL CHECK ("units" > 0),
    "status"       varchar(10) NOT NULL DEFAULT 'brewing',
    "started_at"   timestamptz NOT NULL DEFAULT now(),
    "completed_at" timestamptz
);
CREATE INDEX "ix_batches_beer_id" ON "batches" ("beer_id");
CREATE INDEX "ix_batches_status" ON "batches" ("status");

-- orders ----------------------------------------------------------------------
CREATE TABLE "orders" (
    "id"            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "customer_name" varchar(200) NOT NULL,
    "status"        varchar(10)  NOT NULL DEFAULT 'pending',
    "reject_reason" text         NOT NULL DEFAULT '',
    "created_at"    timestamptz  NOT NULL DEFAULT now(),
    "updated_at"    timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX "ix_orders_status" ON "orders" ("status");

CREATE TABLE "order_lines" (
    "order_id"         uuid    NOT NULL REFERENCES "orders" ("id") ON DELETE CASCADE,
    "beer_id"          uuid    NOT NULL,
    "units"            integer NOT NULL CHECK ("units" > 0),
    "unit_price_cents" bigint  NOT NULL CHECK ("unit_price_cents" >= 0),
    "currency"         char(3) NOT NULL,
    PRIMARY KEY ("order_id", "beer_id")
);
