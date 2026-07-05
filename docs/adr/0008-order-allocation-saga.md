# ADR-0008: Event-sourced order-allocation saga with compensating transactions (book Ch. 12)

- **Status**: accepted (completes the "no compensation yet" caveat of ADR-0002)
- **Date**: 2026-07-05

## Context

A sales order spans several `Availability` aggregates — one local
transaction per beer, no global atomicity (Chapter 12's exact problem).
Before this ADR, each row was allocated independently: a two-row order
could allocate row 1, fail row 2, and leave row 1's stock held forever
with the order none the wiser.

## Decision

An **event-sourced saga** (the book's durable-execution recommendation)
coordinates the allocation inside the warehouses module:

- **Trigger**: the `sales.sales_order_created` integration event starts an
  `OrderAllocationSaga` — an event-sourced aggregate whose stream
  (`OrderAllocationSaga-<orderId>`) records every step outcome, so state
  is always rebuildable after a crash.
- **Steps**: rows allocate sequentially, each a local transaction via
  `UpdateAvailabilityDueToSalesOrder`. Success is the
  `BeerAvailabilityUpdated` event; failure is **`QuantityNotFound`** —
  the book's Figure 12.3 event, recorded in the beer's stream, never a
  silent refusal.
- **Backward recovery**: on failure the saga sends
  `CompensateAvailabilityDueToFailedAllocation` for every
  already-allocated row; `AvailabilityCompensated` restores the stock.
- **Outcome**: `warehouses.order_allocation_completed|rejected`
  integration events; Sales settles the order
  (`SalesOrderAllocated` / `SalesOrderAllocationRejected` on the
  `SalesOrder` aggregate, projected as `allocation_status`).
- **Idempotency** (Ch. 12's hard requirement): all saga `Record*` methods
  are no-ops for facts already observed, compensation commands are sent
  exactly once per transition, and settlement commands re-applied to a
  settled order commit nothing.

Coordination style: **choreographed between contexts** (Sales and
Warehouses only exchange integration events — loose coupling), with the
saga acting as the local coordinator of steps and compensations inside
the warehouse — the hybrid the book recommends when a business process
involves compensating transactions (Table 12.2).

## Consequences

- Partial failures are consistent by construction: the e2e suite proves a
  two-row order that fails on row 2 restores row 1's stock and rejects
  the order.
- The saga's stream is a complete audit of the process — "the movie" of
  the allocation, queryable in the event store.
- Crash resume is possible from the saga stream; an automatic resumer
  (re-driving in-flight sagas at boot) and step timeouts/dead-letter
  queues (Ch. 12's durable-execution extras) are the next ADR when a
  real broker arrives.
- A process manager (central, stateful workflow control) remains
  deliberately out: the book reserves it for branching workflows, which
  this linear compensable sequence is not.
