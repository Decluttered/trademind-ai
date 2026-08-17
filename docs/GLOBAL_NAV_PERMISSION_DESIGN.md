# Global Menu Permission Design (Phase F6)

## Goal

F5 already has page-level `PermissionGuard`; F6 adds **menu-level hiding** — direct access is still intercepted by the Guard / backend.

## Rules

| Role | Menu |
| --- | --- |
| admin | All |
| operator | Actionable modules (no settings sub-items, no user management) |
| readonly | Read-only modules (no write-operation entry menus) |

## Implementation

- `admin/src/utils/menuAccess.ts` — route → permission mapping + `filterMenuByPermission`
- `admin/src/app.tsx` — `menuDataRender` filters the sidebar
- `admin/src/utils/permission.ts` — permission matrix (aligned with the backend `adminperm`)
- `admin/config/routes.ts` — route tree (route names are the menu copy)

## Notes

- Menu filtering **does not replace** backend permission checks
- Settings sub-menus the user lacks permission for (including the Configuration Status Center and Users & Permissions) are hidden from non-admin roles
- `access.ts` remains empty; authorization relies on token + layout + Guard
