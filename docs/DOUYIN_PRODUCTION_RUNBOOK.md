# Douyin Shop Production Runbook

> **Status**: Release Candidate  
> **Real E2E**: `blocked_by_real_credentials`

## Pre-launch Check Order

1. Back up PostgreSQL (`orders` / `order_items`)
2. Start the application: `migrateDouyinOrderIdempotencyIndexes` **automatically** detects duplicate orders; if duplicates are found, startup fails and outputs `sample_ids` (see [`DOUYIN_DUPLICATE_DATA_REPAIR.md`](DOUYIN_DUPLICATE_DATA_REPAIR.md))
3. Settings → Storage → **Test Public Access**
4. Settings → Platform Open Configuration → Douyin Shop → **Production Preflight**
5. Confirm `real_api_enabled` and the order/inventory/product-draft toggles match expectations
6. Confirm the Douyin Shop runtime status is **Normal** (not paused / emergency-disabled)

## Runtime Status Operations

| Status | Meaning | Operation Entry Point |
| --- | --- | --- |
| `normal` | Tasks execute normally | Settings → Platform Open Configuration → Douyin Shop Runtime Status → Resume |
| `paused` | New tasks and write operations paused | Pause tasks (reason required) |
| `emergency_disabled` | All Douyin Shop write APIs disabled in an emergency | Emergency disable (requires confirmation + reason) |

API: `GET/POST /api/v1/platform/douyin/runtime-status/*`

## Stale / Result-Unknown Tasks

| User-facing Message | Internal recoveryStatus | Handling |
| --- | --- | --- |
| Task execution took too long | `stale` | Check the worker/lease; retry manually |
| Platform processing result cannot currently be confirmed | `result_unknown` | Products: verify with `product.detail` first; never call `product.addV2` blindly |
| Requires review before continuing | `recovery_required` | Retry only after manual confirmation |

## Retry Strategy (Phase 10.3)

- All Douyin Shop OpenAPI calls go through `Client.Do` → `ExecuteWithRetry` (up to 3 attempts, exponential backoff with jitter)
- Permission/parameter/configuration errors are **not retried**
- Token refresh is singleflighted per `shopId`
- Business workers **do not nest** HTTP retries

## Failure Recovery

- Order sync: resumes from checkpoint, retries only the failed page
- Inventory sync: uses the `targetStock` captured at task creation time; older-version tasks never overwrite newer ones
- Image upload: already-succeeded images are skipped; only `force=true` forces a re-upload

## Canary Observation Phase (Phase 10.4)

> Executed once real credentials are available, production preflight has completed, and external production approval has been obtained. Remains **Release Candidate** throughout — it must not be marked production-available until manual acceptance is complete.

### Phase G0 — Pre-release (0–2h)

1. Run production preflight from the admin console, or manually call `POST /api/v1/platform/douyin/production-preflight` (`liveTest: true`)
2. Confirm `runtime-status=normal` and that toggles match the Runbook's § Pre-launch Check
3. Back up PostgreSQL; confirm the migration is not blocked by duplicate orders
4. Record the Git SHA and a configuration snapshot (excluding secrets)

### Phase G1 — Read-only Observation (2–24h)

1. Maintainers check read-only entry points: categories, task center, operations dashboard, and Douyin Shop aggregate health
2. Every 4h, check: `GET /health` queue depth, `task-center/summary` Douyin Shop failure count
3. Spot-check operation logs: `douyin.auth.*`, no tokens in plaintext
4. **Prohibited**: performing real platform write operations unless already in G2 with explicit approval

### Phase G2 — Low-volume Write Path (24–72h)

1. Order sync limited to a single test shop, single SKU, small time window
2. Manually execute the controlled write path per [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md)
3. Verify idempotency: duplicate order upserts = 0, duplicate stock deductions = 0
4. Failed tasks → alert scan → manual handling; P0 errors trigger immediate `paused` or `emergency_disabled`

### Phase G3 — Wrap-up

1. Record sanitized release conclusions in the PR or release ticket; do not commit report artifacts to the repository
2. Complete a rollback drill per [`DOUYIN_ROLLBACK_RUNBOOK.md`](DOUYIN_ROLLBACK_RUNBOOK.md) and record the conclusion in the release ticket
3. Still **do not** introduce Prometheus by default; continue using taskcenter + `/health` + dashboards

## Observability (No Prometheus)

| Check | Frequency | Entry Point |
| --- | --- | --- |
| Process and queue | 5min (automated) / on-demand manual | `GET /health` |
| Douyin Shop failed tasks | Hourly | Ops → Failed Task Center; `GET /task-center/failures?keyword=DOUYIN` |
| Alerts | Every 15min (if the scan worker is enabled) | `POST /task-center/alerts/scan` |
| Operations KPIs | Daily | Workbench → Product Operations Dashboard |
| Preflight | Before every release | `POST .../production-preflight` |

## Related Documents

- [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md)
- [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md)
- [`DOUYIN_DUPLICATE_DATA_REPAIR.md`](DOUYIN_DUPLICATE_DATA_REPAIR.md)
- [`DOUYIN_ROLLBACK_RUNBOOK.md`](DOUYIN_ROLLBACK_RUNBOOK.md)
