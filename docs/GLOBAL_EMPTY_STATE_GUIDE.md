# Global Empty State Guidance (Phase F6)

## Component

`admin/src/components/ui/EmptyState.tsx`

- `title` — why it's empty
- `description` — what to do next
- `actionLabel` + `actionPath` — recommended entry point

## Applicable Pages

| Page | Empty-state Notes |
| --- | --- |
| Dashboard | Cards still show 0 + emptyHint when there's no data; the recent activity section guides the user toward collection/configuration |
| Collection Center | Enter a link or configure a collection service |
| Product Drafts | Collect or create manually |
| AI Operations Workbench | Requires pending products or batches first |
| Order List | Configure shop authorization and sync; a seed script is available for the demo |
| Order Exceptions | Generated automatically after order sync |
| Inventory Center | Inventory only exists after listed SKUs are bound |
| Customer Service Center | Authorize the shop and sync messages |
| Failed Task Center | No failures is the normal state; failures display a retry entry point |
| Configuration Status Center | Complete settings one item at a time |
| Users & Permissions | Admin creates users and assigns shops |

## Copy Source

Prefer `PAGE_COPY` and `EMPTY_GUIDE` from `admin/src/constants/copywriting.ts`.
