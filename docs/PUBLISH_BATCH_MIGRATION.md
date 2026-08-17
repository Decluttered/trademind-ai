# Batch Publishing Database Migration (Phase A2.1)

## Overview

Phase A2 extended `product_publish_batches` / `product_publish_tasks` via GORM AutoMigrate. Phase A2.1 adds an **explicit PostgreSQL migration** (executed at Go startup); the production maintenance phase further adds tenant-ownership backfill, used for:

- Making `product_id` nullable on multi-product batches (compatible with `single_product` / `multi_product`)
- Indexes for batch list and subtask queries
- A partial unique index on `idempotency_key` for active batches (prevents concurrent duplicate creation)
- Backfilling a determinable `tenant_id` for historical `product_publish_tasks`, `product_publications`, and `product_publish_batches`

Implementation files: [`backend/internal/database/migrate_publish_batch_a21.go`](../backend/internal/database/migrate_publish_batch_a21.go), [`backend/internal/database/migrate_product_publish_tenant.go`](../backend/internal/database/migrate_product_publish_tenant.go)

## Execution Timing

Invoked at service startup, **after GORM `AutoMigrate` completes** (the model first fills in columns like `idempotency_key`, `batch_id`, then indexes and constraints are created). Safe to run repeatedly (`IF NOT EXISTS`).

## Changes

### 1. `product_publish_batches.product_id` becomes nullable

```sql
ALTER TABLE product_publish_batches ALTER COLUMN product_id DROP NOT NULL;
```

Only executed when `information_schema` still shows `NOT NULL`. Historical single-product batch data is unaffected.

### 2. Query Indexes

| Index Name | Table | Columns | Purpose |
|--------|-----|-----|------|
| `ix_publish_batches_created_at` | product_publish_batches | created_at DESC | Recent batch list |
| `ix_publish_batches_status` | product_publish_batches | status | Filter by status |
| `ix_publish_tasks_batch_id` | product_publish_tasks | batch_id | Subtasks in batch detail |
| `ix_publish_tasks_target_key` | product_publish_tasks | target_key | Target-dimension queries |
| `ix_publish_tasks_batch_status` | product_publish_tasks | batch_id, status | retry-failed / cancel-pending |

### 3. Partial Unique Index on Idempotency Key

**First check** whether duplicate `idempotency_key` values exist among active batches (`status NOT IN ('failed','cancelled')`):

```sql
SELECT idempotency_key, COUNT(*) AS cnt
FROM product_publish_batches
WHERE idempotency_key <> ''
  AND status NOT IN ('failed','cancelled')
GROUP BY idempotency_key
HAVING COUNT(*) > 1;
```

- **No duplicates**: create it

```sql
CREATE UNIQUE INDEX IF NOT EXISTS ux_publish_batches_idempotency_active
 ON product_publish_batches (idempotency_key)
 WHERE idempotency_key <> '' AND status NOT IN ('failed','cancelled');
```

- **Duplicates found**: skip the unique index; requires manual cleanup before the service restarts.

#### Manual Duplicate Cleanup (Example)

Keep the record with the latest `created_at`, and mark the remaining active duplicate batches as `failed` or delete them after merging their subtasks:

```sql
-- Example: view a duplicate group
SELECT id, idempotency_key, status, created_at
FROM product_publish_batches
WHERE idempotency_key = '<duplicate_key>'
ORDER BY created_at DESC;
```

After cleanup, restart the service; the migration will create the unique index automatically.

### 4. Historical Tenant Ownership Backfill

`migrateProductPublishTenant` runs after AutoMigrate completes on the publishing tables, and can be run repeatedly:

- Tasks and publications are backfilled from the existing `products.tenant_id`.
- Single-product batches are backfilled preferentially from their `product_id`; multi-product batches are backfilled from the tenant of their already-backfilled subtasks.
- Only updates rows where `tenant_id=0` and ownership can be determined; orphaned products, orphaned tasks, or historical rows whose ownership cannot be determined remain at `0` — tenancy is never guessed.

Back up the database and verify remaining tenant-zero rows on an isolated PostgreSQL replica before going live. The migration does not delete or reassign historical data that cannot be confirmed.

## Rollback

Run as needed (**rolling back the unique index does not affect data**):

```sql
DROP INDEX IF EXISTS ux_publish_batches_idempotency_active;
DROP INDEX IF EXISTS ix_publish_tasks_batch_status;
DROP INDEX IF EXISTS ix_publish_tasks_target_key;
DROP INDEX IF EXISTS ix_publish_tasks_batch_id;
DROP INDEX IF EXISTS ix_publish_batches_status;
DROP INDEX IF EXISTS ix_publish_batches_created_at;
```

Restore `product_id NOT NULL` (only if confirmed there are no multi_product batches and all rows have a product_id):

```sql
-- Caution: only run when there are no rows with a NULL product_id
ALTER TABLE product_publish_batches ALTER COLUMN product_id SET NOT NULL;
```

## Relationship to GORM AutoMigrate

Column definitions (`batch_type`, `name`, `task_count`, `idempotency_key`, etc.) are still maintained by the GORM model + AutoMigrate. This migration **only** handles the nullable constraint and indexes; it does not replace model evolution.

## Changelog

| Date | Notes |
|------|------|
| 2026-08-14 | Added idempotent tenant-ownership backfill for publish tasks, publications, and batches |
| 2026-06-19 | Phase A2.1 initial migration |
