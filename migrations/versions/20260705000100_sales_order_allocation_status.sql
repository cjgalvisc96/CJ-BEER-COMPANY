-- The order-allocation saga (ADR-0008, book Ch. 12) settles every order:
-- pending → allocated | rejected. The read model projects that status.
ALTER TABLE "sales_orders"
    ADD COLUMN "allocation_status" varchar(20) NOT NULL DEFAULT 'pending',
    ADD COLUMN "rejection_reason" text NOT NULL DEFAULT '';
