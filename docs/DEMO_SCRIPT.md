# Demo Script (Phase R1)

> **Release status**: `MVP Demo Ready`
> Estimated duration: 15–25 minutes (including AI review and bulk listing)

## Prerequisites

1. Local or staging environment is running: `docker compose up -d` + backend + admin panel
2. `.\scripts\seed-demo-data.ps1` has been run (optional, recommended)
3. AI provider is configured (required for a copy dry run; an image provider key is required for white-background images)
4. An admin account can log in

## Standard Demo Path

| Step | Action | Entry point | Expected result |
| --- | --- | --- | --- |
| 1 | Open the AI product operations workbench | AI Tools → Product Operations Workbench `/ai/operation-workbench` | Stat cards and to-do list load |
| 2 | View AI copy pending review | Filter "AI copy pending review" | List shows type/priority |
| 3 | Open copy review and apply one item | Click a to-do → review page `/product/ai-text-batches/:id` | Compare original/suggested text, apply succeeds |
| 4 | Return to workbench and refresh | Click "Refresh to-dos" | The corresponding to-do count decreases |
| 5 | View AI images pending review | Filter "AI images pending review" | Thumbnails and placeholders render correctly |
| 6 | Apply an image to the gallery | Apply from the review page | New image visible under the product detail's Images tab |
| 7 | View product detail operations progress | Product → Draft → progress bar at top of detail page | Steps and blockers displayed |
| 8 | Open publish check | Detail → Publish Check / workbench link | Three states: passed / warning / failed |
| 9 | Select multi-platform, multi-shop listing targets | Listing tab or bulk listing wizard | TikTok / Shopee etc. show "local draft only" |
| 10 | Create a local listing draft | Create at step 5 of the bulk listing wizard | Batch is success or partial_success |
| 11 | View the bulk listing batch | `/product/publish-batches/:id` | Subtask statuses shown, technical details collapsible |
| 12 | View the failed task center | Ops → Failed Task Center `/ops/task-center/failures` | Deep links to copy/image/listing batches |

## Douyin Notes (Demo Boundaries)

- Can demonstrate Douyin configuration, the OAuth entry point, the listing tab, and the "Create Douyin product draft" button
- **Without a real App Key / shop credentials**, create-draft is expected to return `blocked_by_real_credentials`, and this is **not** marked Production Ready
- Direct publishing is **not** demonstrated

## Production Maintenance Acceptance

Automated regression is executed by GitHub Actions. Demo data can be generated ad hoc as needed:

```powershell
.\scripts\seed-demo-data.ps1 -ApiBase https://<pre-api-domain> -OutFile docs/demo-dataset.preprod.json
```

The run output is for that one-time manual acceptance check only and is not committed to Git. Pages, states, and real business flows are manually signed off against [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md).

## Changelog

| Date | Description |
| --- | --- |
| 2026-06-27 | Phase R1.2: notes on staging output file; real HTTPS pending ops onboarding |
| 2026-06-27 | Phase R1.1: corrected the failed task center route to `/ops/task-center/failures` |
| 2026-06-27 | Phase R1 standard demo path |
