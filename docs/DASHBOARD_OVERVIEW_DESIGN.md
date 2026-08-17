# Dashboard Operations Overview Design (Phase F6)

## Positioning

`/dashboard/product-operations` (menu name **Operations Overview**) is the project-wide operations entry point, not a complex BI tool.

- Read-only DB aggregation; does not call external platform APIs
- Does not auto-sync orders / inventory / send customer service messages
- Does not load raw / prompt / platform raw large fields
- Degrades gracefully on a single-module failure, without causing a full-page 500

## API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/dashboard/product-operations` | Full dashboard (KPIs / to-dos / funnel / anomalies / recent activity) |
| GET | `/api/v1/dashboard/overview` | Modular overview + 10 top cards |
| GET | `/api/v1/dashboard/todos` | Unified to-do stream (P0/P1/P2) |
| GET | `/api/v1/dashboard/health` | Subsystem health summary + configuration risk |

## Top Cards (overview.cards)

1. Today's collection tasks
2. Product drafts
3. AI items pending review
4. Publish check issues
5. Listing task anomalies
6. Order anomalies
7. Inventory anomalies
8. Customer service pending replies
9. Failed tasks
10. Configuration risk

Each card: `count` / `status` / `priority` / `link` / `emptyHint`.

## RBAC

- `admin`: full-tenant aggregation
- `operator` / `readonly`: filtered by `user_store_permissions` across orders, customer service, inventory, listings, and products (via platform config / publication association)
- The `shopId` query parameter can be combined with scope

## Module

`backend/internal/modules/operationdashboard`

- `service.go` — aggregation logic
- `overview.go` — overview / todos / health
- `scope.go` — shop scope helpers

## Frontend

- Page: `admin/src/pages/Dashboard/ProductOperations/index.tsx`
- Service: `admin/src/services/dashboard.ts`
- Fallback defaults: `admin/src/constants/dashboardDefaults.ts`
