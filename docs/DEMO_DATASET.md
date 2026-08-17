# Demo Dataset Notes (Phase F7)

> **Release status**: `MVP Demo Ready` (not Production Ready)
> Generation scripts: `scripts/seed-demo-data.ps1` / `scripts/seed-demo-data.sh`
> Permission seeding: `scripts/seed-demo-permissions.ps1`
> Machine-readable output is generated on demand by the seed scripts; run artifacts are not maintained as long-term repository documentation.

## Generation

```powershell
# Requires a local API and an admin account (reads ADMIN_BOOTSTRAP_* from .env)
.\scripts\seed-demo-data.ps1 -ApiBase http://127.0.0.1:8080
.\scripts\seed-demo-permissions.ps1 -ApiBase http://127.0.0.1:8080
```

See [`DEMO_SEEDING_GUIDE.md`](DEMO_SEEDING_GUIDE.md) for details.

The script will:

1. Call `a1-prepare-samples.ps1` to fill out the A1.1 sample matrix (20 slots)
2. Create the R1 extended set of 20 product scenario categories (see table below)
3. Aggregate existing AI copy/image batches, failed tasks, and workbench to-dos
4. Optionally create local listing batches and AI copy seed batches

## Product Samples (20 categories)

| # | tag | Description |
| --- | --- | --- |
| 1 | title_complete | Title complete |
| 2 | title_pending_optimize | Title pending optimization |
| 3 | description_empty | Description empty |
| 4 | description_pending | Description pending optimization |
| 5 | main_images_complete | Main images complete |
| 6 | main_images_missing | Main images missing |
| 7 | detail_images_low | Insufficient detail images |
| 8 | multi_sku | Multiple SKUs |
| 9 | stock_unknown | Stock unknown |
| 10 | price_anomaly | Price anomaly |
| 11 | attributes_missing | Attributes missing |
| 12 | publish_check_passed | Publish check passed candidate |
| 13 | publish_check_warning | Publish check warning |
| 14 | publish_check_failed | Publish check failed |
| 15 | ai_text_pending_review | AI copy pending review |
| 16 | ai_image_pending_review | AI image pending review |
| 17 | ai_conflict | AI conflict |
| 18 | local_publish_draft | Local publish draft |
| 19 | douyin_blocked_credentials | Douyin blocked_by_real_credentials |
| 20 | multi_platform_targets | Multiple platform/shop targets |

Full `productId` values are in `docs/demo-dataset.json` → `productSlots`.

## Task Samples

| Type | Expected status | Source |
| --- | --- | --- |
| AI copy batch | success / partial_success | Existing batches + optional seeding |
| AI image batch | success / partial_success | Existing batches |
| Bulk listing batch | success / partial_success | Existing + local_draft_only seeding |
| Failed task center | failure | Existing failed tasks |
| Product operations workbench | todos | Aggregated to-dos |

## Order Samples (Phase F2 / F7)

| tag | Description |
| --- | --- |
| normal_matched_order | SKU matched, can demo deduction |
| unmatched_sku_order | SKU unmatched, anomaly workbench |
| sync_partial_success | Order sync partial_success (requires shop / mock) |

See `docs/demo-dataset.orders.json` for details.

## Inventory Samples (Phase F3 / F7)

| tag | Description |
| --- | --- |
| normal_stock_sku | Normal stock |
| low_stock_sku | Below warning threshold |
| zero_stock_sku | Zero stock |
| deduct_success_order | Successful deduction path |
| deduct_blocked_unmatched_order | Deduction blocked due to unmatched SKU |

See `docs/demo-dataset.inventory.json` for details.

## Customer Service Samples (Phase F4 / F7)

| tag | Description |
| --- | --- |
| pending_reply | Conversation pending reply |
| ai_suggestion_generated | AI suggestion generated, pending confirmation |
| send_failed | Send failed (best-effort) |

See `docs/demo-dataset.customer-service.json` for details. The AI suggestion step may be skipped if the AI provider is not configured.

## Dashboard KPI Samples (Phase F6 / F7)

After running the seed, probe `GET /dashboard/overview|todos|health`, covering 10 KPIs: collection failures, product drafts, AI pending review, publish checks, listing anomalies, order anomalies, inventory warnings, customer service pending replies, failed tasks, and configuration risk.

See `docs/demo-dataset.dashboard.json` for details.

## Notes

- Douyin's real create-draft flow is still a **Release Candidate**; without credentials, sample #19 is expected to return `blocked_by_real_credentials`
- Bulk listing seeding depends on a `local_draft_only` platform shop; performance regressions are covered by GitHub Actions and manual maintainer acceptance testing.
- Product titles use the ASCII prefix `R1 demo` in the seed script for easy searching; you can rename the display name in the admin panel before a demo

## Changelog

| Date | Description |
| --- | --- |
| 2026-06-30 | Phase F7: order / inventory / customer service / dashboard samples |
| 2026-06-27 | Phase R1.1: seed data and 12-step manual walkthrough |
| 2026-06-27 | Phase R1 initial version: seed script + demo-dataset.json |
