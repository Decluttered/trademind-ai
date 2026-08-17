# Douyin Shop Rollback Runbook

> Rollback drills are performed manually by maintainers in a controlled environment; sanitized conclusions are recorded in the PR or release ticket. Do not commit drill reports, screenshots, or log artifacts to the repository.

## Emergency Disable (Preferred, No Data Deletion)

1. Admin console → Platform Open Configuration → Douyin Shop Runtime Status → **Emergency Disable**
2. Or via API: `POST /api/v1/platform/douyin/runtime-status/emergency-disable` with body `{ "reason": "..." }`
3. Confirm workers no longer call Douyin Shop write APIs; historical tasks/orders remain viewable

## Feature Flag Rollback

Disable the following in the `platform_douyin_shop` settings:

- `real_api_enabled`
- `order_sync_enabled`
- `inventory_sync_enabled`
- `product_publish_enabled`

## Database Index Rollback (Indexes Only, No Business Data Deletion)

```sql
DROP INDEX IF EXISTS ux_orders_shop_platform_ext_order;
DROP INDEX IF EXISTS ux_order_items_order_ext_item;
```

## Runtime Status Recovery

```sql
-- Or restore to normal via the admin console/API
UPDATE settings SET item_value = 'normal'
WHERE group_key = 'platform_douyin_shop' AND item_key = 'platform_runtime_status';
```

## Migration Failure Due to Duplicate Data

See [`DOUYIN_DUPLICATE_DATA_REPAIR.md`](DOUYIN_DUPLICATE_DATA_REPAIR.md) — **do not delete automatically**; restart the migration only after manual remediation.

## Verifying the Rollback

- New Douyin Shop tasks should be blocked or marked `cancelled`
- No tokens/secrets in plaintext in the logs
- The application starts normally
