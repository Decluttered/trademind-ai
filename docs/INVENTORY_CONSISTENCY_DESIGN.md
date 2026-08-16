# Inventory Consistency Design (P2)

> Local inventory relies on an **append-only ledger + order-effect table + versioned sync tasks** to guarantee auditability, retryability, and deduplication.

## Core Tables

### `inventory_change_logs` (Ledger)

Append-only audit trail recording every local inventory change:

| Field | Description |
| --- | --- |
| `change_type` | `manual_adjust` / `order_deduct` / `platform_sync`, etc. |
| `before_stock` / `after_stock` / `delta` | Stock before/after the change and the delta |
| `ref_order_id` / `ref_order_item_id` | Order association |
| `business_event_key` | **Globally unique** (`uniqueIndex`), prevents duplicate ledger entries |

### `order_inventory_effects` (Order Deduction Effects)

Unique per `(order_item_id, product_sku_id, effect_type)` (`ux_oie_item_sku_effect`):

- `deduct` / `restore`; status `pending` / `success` / `failed` / `skipped`.
- Successful rows reference `inventory_change_log_id`.
- Re-requesting a row that already succeeded → **skipped** (idempotent).

## `business_event_key` Convention

Aligned with the `idempotency` key; set before writing to the ledger:

| Scenario | Recommended format |
| --- | --- |
| Order deduction | `inventory-deduct:{orderId}:{orderItemId}:{skuId}` |
| Order rollback | `inventory-restore:{orderId}:{orderItemId}:{skuId}` |
| Manual adjustment | `inventory-adjust:{skuId}:{changeLogId}` or a timestamp-based batch number |
| Platform push | `inventory-push:{shopId}:{skuId}:{stockVersion}` |

The unique constraint guarantees that worker retries or duplicate API submissions never produce a second ledger entry.

## Order Deduction Flow

`DeductInventoryForOrder`:

1. Iterate over order lines, applying row-level `SELECT FOR UPDATE` per SKU.
2. Check whether `order_inventory_effects` already has a `success` row.
3. Insufficient stock → write a `failed` effect; no partial deduction (negative-stock policy is configurable).
4. Success → write `inventory_change_logs`, update `product_skus.stock`, and write a `success` effect.

Automatic deduction can be triggered after order sync completes; failures go to the order exceptions workbench under `inventory_deduct_failed`.

## Version Sync (`stockVersion`)

Outbound `inventory_sync_tasks`:

- `input.stockVersion` = `targetStock` (current target-stock snapshot).
- `idempotency.InventoryPush(shopId, skuId, stockVersion)` serves as the business idempotency key.
- `lock_version` participates in worker claiming to prevent concurrent duplicate pushes.

### Pending/Running Deduplication

When creating sync tasks in bulk, `blockingPublicationSKUSet` detects existing pending/running tasks for the same `publication_sku_id`:

- By default it **skips** and records `duplicate_pending_running_task`;
- `force=true` can force creation of a new task (for operational scenarios).

The Douyin path additionally requires SKU binding to be complete (`inventorySyncReady`).

## Platform Push Side Effects

After `ProcessQueuedTask` succeeds or fails, `appendChange` writes to the ledger; Douyin and other metrics are counted via `douyinmetrics`.

## Boundary with Order Sync

- Order upsert does **not** automatically change local SKU stock (except when an explicit policy triggers a deduction).
- Inventory push does **not** write back to order status; the two sides remain traceable via the `order_inventory_effects` association.

## Acceptance Criteria

Duplicate deductions are skipped; `business_event_key` uniqueness prevents double-booking; pending pushes are deduplicated by default. See `POST /api/v1/inventory-sync/tasks/:id/retry` for API retry behavior.
