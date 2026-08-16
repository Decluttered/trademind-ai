# Order Sync partial_success UX (Phase F2)

## Semantics

- **partial_success**: some pages were successfully written to local orders, some pages failed to fetch
- Must not be displayed as "fully successful" or "fully failed"

## Task Output Fields

```json
{
  "totalPages": 10,
  "successPages": 8,
  "failedPages": 2,
  "totalFetched": 120,
  "createdOrders": 5,
  "updatedOrders": 115,
  "matchedItems": 100,
  "unmatchedItems": 20,
  "pageErrors": [{ "page": 3, "error": "..." }]
}
```

## Retry Strategy

- `POST /api/v1/order-sync/tasks/:id/retry` automatically sets `input.retryPagesOnly` for `partial_success`
- Only re-fetches the pages listed in `pageErrors`; successful pages are not repeated

## UI (`/orders/sync-tasks`)

- The detail drawer shows pagination stats and a table of failed pages
- "Retry Failed Pages" button
- Links to the failed task center and the exception workbench

## Failed Task Center

- `partial_success` is included in the failed list (`normalizedStatus=partial_success`)
- `Retryable=true`
