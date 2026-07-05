# Message Catalog

Commands use the producer-consumer pattern (exactly one handler); events
use pub/sub. Topics: `commands.<name>`, `events.<name>`,
`integrationevents.<name>`.

## Commands (imperative)

| Command | Module | Effect |
|---|---|---|
| `sales.create_sales_order` | sales | Create a SalesOrder (aggregateId = SalesOrderId) |
| `warehouses.update_availability_due_to_production_order` | warehouses | Add produced quantity to a beer's availability |
| `warehouses.update_availability_due_to_sales_order` | warehouses | Allocate ordered quantity from a beer's availability |

## Domain events (past tense, stay inside their module)

| Event | Raised by | Consumed by (same module) |
|---|---|---|
| `sales.sales_order_created` | SalesOrder aggregate | read-model projection; integration publisher |
| `warehouses.availability_updated_due_to_production_order` | Availability aggregate | availability projection (quantity = new total) |
| `warehouses.beer_availability_updated` | Availability aggregate | availability projection (quantity = remaining); integration publisher |

## Integration events (cross-context, consumer-driven contracts)

| Event | Producer | Consumers |
|---|---|---|
| `sales.sales_order_created` | sales | **warehouses**: one `UpdateAvailabilityDueToSalesOrder` per row |
| `warehouses.beer_availability_updated` | warehouses | **sales**: notified that stock was allocated |

## The flow (the book's Figure 4.2, implemented part)

```
Sales                     Warehouses                    (Payment)   (Shipping)
  │ SalesOrderCreated ───────▶ allocate stock per row       future      future
  │                           │ BeerAvailabilityUpdated
  │ ◀──────────────────────────┘ (remaining quantity)
```

Allocation beyond availability is a business refusal: the warehouse logs
`warehouses.allocation_refused`, commits nothing, and acks the message
(it is not a poison message).
