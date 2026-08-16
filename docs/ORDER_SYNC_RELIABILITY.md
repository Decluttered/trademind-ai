# Order Sync Reliability Design (P2)

> Order sync idempotency relies on **database unique constraints + upsert semantics + shared webhook rules**, implemented in `ordersync`, `order/sync_platform.go`, and the Phase 10.2 index migration.

## Unique Keys and Indexes

### Order Header (PostgreSQL Partial Unique Index)

```sql
CREATE UNIQUE INDEX ux_orders_shop_platform_ext_order
 ON orders (shop_id, platform, external_order_id)
 WHERE external_order_id IS NOT NULL AND external_order_id <> '' AND deleted_at IS NULL;
```

Business semantics: **the same shop + platform + platform order number** maps to only one local order.

### Order Items

```sql
CREATE UNIQUE INDEX ux_order_items_order_ext_item
 ON order_items (order_id, external_item_id)
 WHERE external_item_id IS NOT NULL AND external_item_id <> '';
```

A duplicate-data check runs before migration; if duplicate groups exist, the migration is **blocked** and points to `DOUYIN_DUPLICATE_DATA_REPAIR.md`.

## Upsert Flow

`order.Service.UpsertSyncedOrders(shopID, platform, payloads)`:

1. Look up the existing row by `(shop_id, platform, external_order_id)`.
2. **Not found** → `Create` the order + `replaceSyncedChildren` (items/shipments).
3. **Found** → update header fields (status, amount, timestamps, `raw_data`); **preserve the operations note** `remark`.
4. Child rows are reconciled by `external_item_id` to avoid duplicate inserts.

Returned metrics: `created` / `updated` / `success` / `failed`; the `ordersync` task output includes `upsertSuccess`, `upsertFailed`.

## Idempotency Key (Unified Idempotency Service)

```text
idempotency.OrderSync(platform, shopID, platformOrderID)
→ order-sync:{platform}:{shopId}:{platformOrderId}
```

Used for cross-request duplicate prevention: syncing the same platform order twice does not produce a second `orders` row.

## Order Sync Task Reliability

- Queue: `ORDER_SYNC_QUEUE_*`; worker DB lease documented in `ordersync/lease.go`.
- Pagination: on `partial_success`, only failed pages are retried (`RetryPages`).
- After sync, `DeductInventoryForOrder` is optionally called; deduction failures are logged but do not block the order upsert success count.
- Platform provider errors are mapped to codes such as `DOUYIN_ORDER_*` and surfaced in the failed task center.

## Shared Webhook Rules

Inbound webhooks (`webhook.Service.Ingest`) share the same duplicate-prevention approach as order sync:

| Layer | Mechanism |
| --- | --- |
| Domain table | `webhook_events (platform, event_id)` unique; `ON CONFLICT DO NOTHING` |
| Idempotency table | `scope=webhook`, `key=webhook:{platform}:{eventId}` |
| ACK | Duplicate events return `duplicate=true`, returning the existing `status` immediately |

When there is no `eventId`, one is derived from the payload's SHA-256. Payload size is capped at **1 MB**.

When order-related webhooks are later processed asynchronously, they should use the same `external_order_id` key space as `UpsertSyncedOrders`.

## Task State and Resumption

- `partial_success`: `pageErrors` records the failed pages; reaching `maxPages` while `hasMore` is still true also counts as partial success.
- Checkpoint fields: `totalFetched`, `createdOrders`, `updatedOrders`, `nextPage`, etc.
- Retry API: `POST /api/v1/order-sync/tasks/:id/retry` (requires `SafeRetry`).

## Acceptance Criteria

1. Syncing the same `external_order_id` twice in a row → only one order, with the `updated` count incrementing.
2. Two workers syncing the same order concurrently → the unique index + transaction guarantee no duplicate rows.
3. Duplicate webhook delivery → `duplicate=true`, `webhook_events` is not written twice.
4. Deliberately leaving duplicate data before migration → startup fails fast and reports sample IDs.
## P3.2 Douyin Webhook Tenant Update

Douyin order webhooks no longer infer a shop by "the only authorized shop". The webhook event must already contain resolver output: `tenant_id`, `internal_shop_id`, `platform_shop_id`, `app_id`, and optional `binding_id`. `DouyinOrderWebhookHandler` validates that scope before calling `UpsertPlatformOrder`, and order lookup/upsert now includes `TenantID` plus `shop_id`.

Webhook domain uniqueness is `webhook_events(platform, tenant_id, platform_shop_id, event_id)`. The older `platform + event_id` explanation remains only as P2.2 historical foundation text.
