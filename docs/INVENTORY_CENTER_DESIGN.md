# Inventory Center Design (Phase F3)

## Routes

| Path | Description |
| --- | --- |
| `/inventory` | Inventory Center main entry (SKU-level summary) |
| `/inventory/alerts` | Inventory alerts |
| `/inventory/deductions` | Inventory deduction records (the former `/inventory/effects` redirects here) |
| `/inventory/sync-tasks` | Platform inventory sync tasks |
| `/inventory/sync-batches` | Sync batches |
| `/inventory/logs` | Local inventory transaction log |

## API

- `GET /api/v1/inventory` — Inventory Center list (keyword / stockStatus / skuBindStatus / syncStatus / hasException, etc.)
- `GET /api/v1/inventory/alerts` — Alert list
- `GET /api/v1/inventory/effects` — Deduction/rollback effect records
- `GET /api/v1/inventory-sync/tasks` — Sync tasks

## Fields

Each row in the Inventory Center corresponds to one **local product_sku**, showing:

- Local stock / available stock (equivalent in the MVP)
- Alert threshold, stock status
- SKU binding status (bound / unbound / ambiguous / none)
- Platform sync status summary
- Most recent deduction, most recent sync, exception count

## Principles

- Platform inventory is not synced automatically
- No automatic restocking, no automatic purchase order creation
- Technical fields are collapsed by default (detail drawer / Technical Details tab)
- All statuses are localized (Chinese-facing UI text)

## Deep Links

- Order detail `?tab=inventory` → Inventory Impact tab
- `/inventory?skuId=:skuId` → locate a SKU
- `/inventory/deductions?orderId=:orderId` → order deduction records
- `/inventory/sync-tasks?productSkuId=:skuId&id=:taskId` → sync task

## Permissions (F3 Lightweight RBAC)

- `admin` / `operator`: can view and perform write operations
- `readonly`: view only; the backend rejects adjust-stock / sync / retry
