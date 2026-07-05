# Message Catalog

Commands use the producer-consumer pattern (exactly one handler); events
use pub/sub. Topics: `commands.<name>`, `events.<name>`,
`integrationevents.<name>`.

## Commands (imperative)

| Command | Module | Effect |
|---|---|---|
| `sales.create_sales_order` | sales | Create a SalesOrder (aggregateId = SalesOrderId) |
| `sales.mark_sales_order_allocated` | sales | Settle the order after the saga completed |
| `sales.mark_sales_order_allocation_rejected` | sales | Settle the order after the saga failed |
| `warehouses.update_availability_due_to_production_order` | warehouses | Add produced quantity to a beer's availability |
| `warehouses.update_availability_due_to_sales_order` | warehouses | One saga step: allocate a row's quantity |
| `warehouses.compensate_availability_due_to_failed_allocation` | warehouses | Compensating transaction: give an allocated quantity back |

## Domain events (past tense, stay inside their module)

| Event | Raised by | Consumed by (same module) |
|---|---|---|
| `sales.sales_order_created` | SalesOrder | order projection; integration publisher |
| `sales.sales_order_allocated` | SalesOrder | status projection |
| `sales.sales_order_allocation_rejected` | SalesOrder | status projection |
| `warehouses.availability_updated_due_to_production_order` | Availability | availability projection |
| `warehouses.beer_availability_updated` | Availability | availability projection; **saga** (step succeeded) |
| `warehouses.quantity_not_found` | Availability | **saga** (step failed — the book's Fig. 12.3 event) |
| `warehouses.availability_compensated` | Availability | availability projection; **saga** (compensation landed) |
| `warehouses.order_allocation_*` (started / step_succeeded / step_failed / step_compensated / completed / rejected) | OrderAllocationSaga | the saga's own event-sourced state (ADR-0008) |

## Integration events (cross-context, consumer-driven contracts)

| Event | Producer | Consumers |
|---|---|---|
| `sales.sales_order_created` | sales | **warehouses**: starts the order-allocation saga |
| `warehouses.order_allocation_completed` | warehouses | **sales**: mark the order allocated |
| `warehouses.order_allocation_rejected` | warehouses | **sales**: mark the order rejected (with reason) |

## The saga flow (book Ch. 12)

```
Sales                                Warehouses
  │ SalesOrderCreated ───────────────▶ OrderAllocationSaga starts
  │                                    │ allocate row 1 ── BeerAvailabilityUpdated ─┐
  │                                    │ ◀──────────────────────────────────────────┘
  │                                    │ allocate row 2 ── QuantityNotFound ─┐
  │                                    │ ◀────────────────────────────────────┘
  │                                    │ COMPENSATE row 1 ── AvailabilityCompensated
  │ ◀── order_allocation_rejected ─────┘ (or _completed when every row allocates)
  │ order status → rejected | allocated
```

Every message in the saga is idempotent under redelivery; a shortage is a
recorded business fact (`QuantityNotFound`), while infrastructure failures
propagate for redelivery by the bus.
