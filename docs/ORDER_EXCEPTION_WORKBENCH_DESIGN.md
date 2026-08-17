# Order Exception Workbench Design (Phase F2)

## Route

`/orders/exceptions?orderId=&exceptionType=`

## Exception Types

| Type | Source |
| --- | --- |
| sku_unmatched | `product_sku_id` not bound |
| sku_ambiguous | Multiple candidates pending confirmation |
| insufficient_stock / inventory_deduct_failed | Inventory-affecting ledger entry |
| inventory_sync_failed | Inventory sync task failed |
| order_sync_partial_failed | `order_sync_tasks.status=partial_success` |

## Deep Links

| Direction | URL |
| --- | --- |
| Exception → Order Detail | `/orders/:orderId` |
| Exception → Sync Task | `/orders/sync-tasks?id=:taskId` |
| Exception → Failed Task Center | `/ops/task-center/failures?taskType=order_sync&keyword=:id` |
| Order Detail → Exception | `/orders/exceptions?orderId=:id` |

## Mark as Handled

- **Does not equal** automatically fixing the SKU or inventory facts
- Only writes to `order_exception_marks`; the UI copy states this explicitly

## Manual Binding

- Requires a confirmation dialog
- Low-confidence candidates are not auto-bound
- Does not overwrite `manual_bound`
