# Douyin Shop Duplicate Order Data Repair Guide (Phase 10.3)

> **Purpose**: Manually identify and handle historical duplicate data before creating the `ux_orders_shop_platform_ext_order` / `ux_order_items_order_ext_item` unique indexes.
> **Principle**: No automatic deletion, no silent merging; all cleanup requires manual confirmation.

---

## 1. Pre-Launch Check Order

1. Back up PostgreSQL (the full database or the `orders` / `order_items` tables).
2. Start the application or run the database migration; `migrateDouyinOrderIdempotencyIndexes` will **automatically** detect duplicate data (no manual SQL needed).
3. If the migration succeeds → the unique indexes have been created or already exist, and you can proceed with the launch.
4. If startup/migration reports `phase102 blocked: found ... duplicate ...` → **stop the launch**, resolve it manually per this document, then retry.

---

## 2. Duplicate Order Rules (orders)

For multiple records sharing the same `(shop_id, platform, external_order_id)`:

| Retention Priority | Condition |
| --- | --- |
| 1 | Has complete `order_items` and the associated inventory deduction has occurred |
| 2 | Has the most recent `updated_at` and a status closer to a terminal state (shipped/completed) |
| 3 | Has the earliest `created_at` and the most complete data fields |

**Do not keep**:

- Records with an empty `external_order_id` or that are obviously test data
- Copies that are clearly duplicate imports with large numbers of missing fields

**How to handle**:

- Soft-delete (`deleted_at`) the discarded records, or merge fields into the retained record and then soft-delete the copies
- Log the action: operator, retained ID, list of discarded IDs, reason

---

## 3. Duplicate Order Items (order_items)

For duplicates sharing the same `(order_id, external_item_id)`:

- Keep the one record consistent with the platform SKU, quantity, and amount
- Delete (soft-delete) the remaining copies
- Confirm that inventory deduction logs (`inventory_change_logs`) reference only the retained row

---

## 4. Error Message on Migration Failure

If the application migration detects duplicates, it returns something like:

```text
phase102 blocked: found N duplicate order groups (example shop=... platform=... external_order_id=... count=... sample_ids=[...])
```

- `sample_ids` are internal UUIDs and contain no buyer privacy data
- Query the full records by sample ID, then make a manual decision

---

## 5. Rollback

Index rollback (does not affect business data):

```sql
DROP INDEX IF EXISTS ux_orders_shop_platform_ext_order;
DROP INDEX IF EXISTS ux_order_items_order_ext_item;
```

---

## 6. Post-Repair Verification

1. The duplicate-check SQL returns 0 rows
2. Restart the application and confirm the migration succeeds
3. Trigger one order sync and confirm the upsert does not error
4. After repeated syncs of the same `external_order_id`, the database still contains only 1 record

---

**Status**: Release Candidate; verification that the duplicate count is 0 in a real environment is still pending in a credentialed environment.
