# Event Catalog

Topics are the public contract between contexts. Consumers deserialize their
own local structs (consumer-driven contracts) — they never import the
producer's types.

## Domain events

| Topic | Producer | Consumers | Payload (beyond `occurred_at`) |
|---|---|---|---|
| `catalog.beer_created` | catalog | — (observability) | `beer_id`, `name`, `style` |
| `catalog.beer_price_changed` | catalog | — | `beer_id`, `old_price_cents`, `new_price_cents`, `currency` |
| `catalog.beer_retired` | catalog | — | `beer_id`, `name` |
| `brewing.batch_started` | brewing | — | `batch_id`, `beer_id`, `units` |
| `brewing.batch_completed` | brewing | **inventory** (replenishes stock) | `batch_id`, `beer_id`, `units` |
| `inventory.stock_replenished` | inventory | — | `beer_id`, `units`, `quantity` |
| `inventory.stock_reserved` | inventory | — | `beer_id`, `units`, `quantity` |
| `inventory.stock_level_low` | inventory | — (alerting hook) | `beer_id`, `quantity`, `reorder_level` |
| `orders.order_placed` | orders | **inventory** (reserves stock) | `order_id`, `customer_name`, `lines[]{beer_id,units}`, `total_cents`, `currency` |
| `orders.order_confirmed` | orders | — | `order_id` |
| `orders.order_rejected` | orders | — | `order_id`, `reason` |
| `orders.order_cancelled` | orders | — | `order_id` |

## Integration events (process outcomes, not aggregate facts)

| Topic | Producer | Consumers | Payload |
|---|---|---|---|
| `inventory.order_stock_reserved` | inventory | **orders** (confirms order) | `order_id` |
| `inventory.order_stock_rejected` | inventory | **orders** (rejects order) | `order_id`, `reason` |

## The order-fulfilment choreography

```
orders                          inventory
  │ orders.order_placed ───────────▶ verify all lines, then reserve
  │                                  │
  │ ◀── inventory.order_stock_reserved (all lines held)
  │     → Order.Confirm()
  │ ◀── inventory.order_stock_rejected (any line short / untracked)
  │     → Order.Reject(reason)
```

Race note: an order can be cancelled by the customer while the reservation
is in flight. The orders event handlers treat `ErrOrderNotPending` as an
expected outcome (logged, acked) — not a processing failure.
