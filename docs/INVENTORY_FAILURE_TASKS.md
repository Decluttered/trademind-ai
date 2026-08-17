# Inventory Failure Task Integration (Phase F3)

## Failed Task Center `taskType`

`inventory_sync` — source table `inventory_sync_tasks`

## Failure Categories

| Category | Description |
| --- | --- |
| inventory_deduct_failed | Order deduction failed (exceptions workbench is the primary entry point) |
| inventory_sync_failed | Sync task failed |
| inventory_sync_partial_success | Some SKUs failed to sync |
| inventory_sku_not_bound | Blocked because SKU is not bound |
| inventory_sku_ambiguous | Blocked because binding is ambiguous |
| inventory_stock_invalid | Invalid stock value |
| inventory_platform_permission_denied | Insufficient platform permissions |
| inventory_product_not_bound | Platform product not bound |
| inventory_platform_sku_missing | Platform SKU missing |

## Deep Links

- Task detail → `/inventory/sync-tasks?id=:taskId`
- Exceptions workbench inventory_sync_task → TaskCenterURL + InventoryURL
- Order deduction failure → `/inventory/deductions?orderId=:orderId`

## Actions

View inventory, view product, view order, view SKU binding, retry sync, view sync task.

## Principles

- Blocks (not bound / ambiguous) are not counted as system failures; they are surfaced with a separate notice
- Sync failures and low-stock alerts are displayed separately
- Once resolved, status is kept in sync with the task/exception mark
