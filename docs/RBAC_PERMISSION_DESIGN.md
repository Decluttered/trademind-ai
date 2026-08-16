# RBAC Permission Design (Phase F5)

## Roles

| Role | Identifier | Description |
| --- | --- | --- |
| Admin | `admin` | Full system configuration, user management, all shop data |
| Operator | `operator` | Product/order/inventory/customer-service/task operations within authorized shops |
| Read-only | `readonly` | Read-only within the authorized scope; write operations are blocked on the backend |

## Permission Matrix

Implementation location: `backend/internal/pkg/adminperm/matrix.go`
Profile export: `GET /api/v1/auth/profile` → `permissions[]`

## Error Codes

| code | Meaning |
| --- | --- |
| 40302 | No module permission |
| 40303 | No shop permission |
| 40304 | Write operation on read-only account |
| 40305 | System configuration permission required |
| 40306 | User management permission required |

## Backend Enforcement

- Unified package: `backend/internal/pkg/adminperm`
- Write operations must be validated in the handler/service layer, not merely relying on hidden frontend buttons
- Shop detail / deep links: return **404** when unauthorized (does not leak resource existence)

## Frontend

- `admin/src/utils/permission.ts`
- `admin/src/hooks/usePermission.ts`
- `admin/src/components/PermissionGuard.tsx`
