# Douyin Shop Order Webhook Unified Upsert Design

## Unified Entry Point

```go
order.Service.UpsertPlatformOrder(ctx, PlatformOrderUpsertInput)
```

`source` values: `polling` | `webhook` | `manual_sync` | `reconciliation`

## Flow

```text
Webhook Receiver → verify signature → Ingest → ProcessEvent
  → HandleDouyinPlatformEvent → MapDouyinOrderWebhookEvent
  → UpsertPlatformOrder → MatchOrderItemsForOrder (side effect, idempotency-protected)
```

Polling sync:

```text
ordersync.ProcessQueuedTask → ToSyncedPayloads → UpsertPlatformOrders(source=polling)
```

## Unique Key

`shop_id + platform + external_order_id` (tenant field reserved on `Order.TenantID`)

## Forbidden

- The webhook handler writing directly to the `orders` table
- Bypassing `idempotency.Service`
- Maintaining separate logic paths for webhook vs. polling writes

## P3.2 Tenant/Shop Scope

Order webhooks must use resolver output already stored on the webhook event. `HandleDouyinOrderEvent` validates `tenantId`, `internalShopId`, and `platformShopId` before unified upsert. `UpsertPlatformOrder` receives both `TenantID` and `PlatformShopID`; raw order summaries may include non-sensitive platform shop identifiers, but never secrets or tokens.

The webhook handler must not write `orders` directly, bypass `idempotency.Service`, fork polling/webhook upsert logic, or infer the shop from "the only authorized shop".
