# Inventory Sync UX (Phase F3)

## `inventory_sync_enabled` Onboarding

Defaults to **off**. Display consistent copy in the following locations:

> Platform inventory sync is not currently enabled for this shop. Once enabled, the system can sync local inventory to the platform after manual confirmation.

Display locations:

- Inventory center banner
- Inventory sync tasks page
- Product detail → Inventory tab (existing platform configuration hint)
- Platform integration settings (`inventory_sync_enabled` toggle)

Actions: **Go to settings** → `/settings/platforms`; **View documentation** → `/settings/inventory`

## SKU Blocking (Standard Copy)

| Scenario | Copy |
| --- | --- |
| Not bound | This SKU is not yet bound to a platform SKU, so platform inventory cannot be synced. Please complete the SKU binding before retrying. |
| Ambiguous | This SKU has multiple possible matches. Manual confirmation is required before inventory sync can continue. |

Applies to: order deduction, platform sync, product inventory tab, inventory center, alerts, failed tasks, exceptions workbench.

## Sync Task Status (User-Visible)

Pending / Running / Success / Partial Success / Failed / Cancelled / Blocked

Blocking reasons: SKU not bound, binding conflict, platform product not created, platform SKU missing, sync toggle not enabled, insufficient permissions, invalid stock value.

## Retry

- Failed and partially successful tasks can be retried manually
- Retries preserve idempotency protection (new task or batch retry API)
- Does not call the real platform OpenAPI (Demo / RC environments)
