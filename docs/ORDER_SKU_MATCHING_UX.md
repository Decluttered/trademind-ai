# Order SKU Matching and Candidates UX (Phase F2)

## States

| match_status | Meaning |
| --- | --- |
| matched | Matched |
| manual_bound | Manually bound |
| ambiguous | Candidates pending confirmation |
| unmatched | Unmatched |
| skipped | Skipped |

## Candidate API

- `GET /api/v1/order-items/:itemId/sku-candidates`
- `POST /api/v1/orders/:id/sku-candidates/batch`

Candidate fields: confidence, reason, matchSignals, stock, sourceBreakdown.

## Rules

1. Candidates are read-only; **not auto-bound**
2. Binding goes through `POST /order-items/:itemId/bind-sku` or the bind-sku action in the exception workbench
3. Low-confidence candidates (<40) are labeled "Reference" in the UI
4. Rows already `manual_bound` are not overwritten by automatic matching
5. Bind/unbind actions write to the operation log

## Page Entry Points

- Order Detail "SKU Matching" tab
- `/orders/sku-matches` global list
- Exception workbench "View Candidates / Bind SKU"

## Deep Link

`/orders/:orderId?itemId=:orderItemId` opens the detail view and focuses the corresponding row
