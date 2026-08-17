# Inventory Deduction Records (Phase F3)

## Route

`/inventory/deductions` (`/inventory/effects` 301-style redirect)

## Data Source

The `order_inventory_effects` table, joined with orders / product_skus.

## Fields

| Field | Description |
| --- | --- |
| Deduction time | createdAt |
| Source order | links to `/orders/:id?tab=inventory` |
| Product / SKU | linked product title and skuCode |
| Deducted quantity | quantity |
| Stock before/after deduction | beforeStock / afterStock |
| Deduction status | success / failed / skipped |
| Failure reason | errorMessage |

## Source Types

- `deduct` → deduction from order sync
- `restore` → system rollback
- Manual corrections go through `inventory_change_logs` (the inventory ledger page)

## Related Flows

- Deduction failure → order exceptions workbench, `inventory_deduct_failed`
- Order detail inventory tab → deep link to deduction records
- Failed task center (inventory-sync tasks are categorized separately)

## Out of Scope

- Silent deduction failures
- Multi-warehouse WMS ledger (not part of the MVP)
