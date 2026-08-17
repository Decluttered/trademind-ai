# Douyin Shop Order Sync Design

## API

- List: `order.searchList` (paginated, page/size, Unix timestamps)
- Detail: `order.orderDetail` (shop_order_id)

## Sync Cursor (DouyinSyncCursor)

Each shop + syncType maintains one cursor row:

```
(shop_id, sync_type) UNIQUE
fields: cursor, window_start, window_end, last_success_at, version
```

`UpsertDouyinCursor`: uses ON CONFLICT to ensure the version only advances forward (never rolls back).

## Sync Strategy

| Mode | Description |
|------|------|
| window | Fetches by time window, up to maxPages pages per run |
| cursor | Incremental fetch based on the previous cursor |

## Field Mapping

| Douyin Field | PlatformOrder Field | Description |
|------------|-------------------|------|
| order_id | ExternalOrderID | Platform order ID |
| order_status | Status | Mapped via MapOrderStatus |
| pay_amount | TotalAmount | In cents, converted to yuan |
| create_time | OrderedAt | Unix seconds |

## Error Handling

| Error Type | Handling |
|---------|------|
| DOUYIN_ORDER_RATE_LIMITED | Wait for Retry-After, then retry |
| DOUYIN_AUTH_EXPIRED | Trigger token refresh / re-authorization |
| DOUYIN_ORDER_PARSE_FAILED | Log the raw payload, skip the current order |
