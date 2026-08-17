# Order Center Design (Phase F2)

> Routes follow the Admin implementation: `/orders/list`, `/orders/:id`, `/orders/exceptions`, `/orders/sync-tasks`.

## Goals

- Order list displays SKU matching, inventory deduction, sync, and exception summaries
- Standalone order detail deep link `/orders/:id?itemId=`
- Cross-navigation with the exceptions workbench, failed task center, and sync tasks

## List Fields

| Field | Source |
| --- | --- |
| SKU matching status | aggregated from `order_item_sku_matches` |
| Inventory deduction status | aggregated from `order_inventory_effects` |
| Sync status | derived from `externalOrderId` + `platform` |
| Exception count | unmatched/ambiguous + deduction failures |

## Permissions (F2 lightweight)

| Role | Capability |
| --- | --- |
| admin / operator | View, bind SKU, retry |
| readonly | View only; write-operation APIs return 403 |

## Sensitive Information

- Detail API masks phone numbers and emails by default
- Platform `rawData` is not returned in the list

## Related Documents

- [ORDER_EXCEPTION_WORKBENCH_DESIGN.md](ORDER_EXCEPTION_WORKBENCH_DESIGN.md)
- [ORDER_SYNC_PARTIAL_SUCCESS_UX.md](ORDER_SYNC_PARTIAL_SUCCESS_UX.md)
- [ORDER_SKU_MATCHING_UX.md](ORDER_SKU_MATCHING_UX.md)
