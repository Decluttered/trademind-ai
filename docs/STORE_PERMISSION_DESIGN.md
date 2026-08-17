# Store Authorization Design (Phase F5)

## Data Table

`user_store_permissions`

| Field | Description |
| --- | --- |
| userId | Admin UUID |
| storeId | Store UUID (`shops.id`) |
| platform | Platform slug (redundant, for display purposes) |
| permissionScope | `view` / `operate` / `manage` |

## Scoping Rules

- **admin**: no store filtering
- **operator / readonly**: lists and detail views are filtered by `storeId IN (...)`
- Stores without authorization: list is empty; detail view returns 404

## Covered Modules

Orders, customer service, inventory sync/alerts, failed task center, operation logs (records that include a `shopId`)

## API

- `PUT /api/v1/admin/users/:id/store-permissions` (admin only)
