# Multi-Platform Publishing Design (Phase A2)

> Single-product multi-platform / multi-shop publishing center (A1.2) + **bulk creation of publish drafts for multiple products** (A2).

## Platform Capability Tiers

| Internal capability code | Operator-facing copy | Behavior |
| --- | --- | --- |
| `real_draft_create` | Can create a platform draft | Calls the platform's real write API to create a draft (currently **Douyin** only) |
| `local_draft_only` | Generates a local draft only | Generates a local `product_publications` entry + task snapshot; **does not call** the external platform API |
| `not_configured` | Not yet configured | Platform open-API configuration or publishing preset is incomplete |
| `not_authorized` | Shop not authorized | OAuth must be completed in shop management first |
| `disabled` | Disabled | Provider or capability has been turned off |

Capability source: the `GET /api/v1/platform/providers` registry + shop table + completeness of the platform open-API configuration; **the platform list is never hardcoded in the page**.

## Single-Product Multi-Target Publishing (A1.2)

1. **Publish targets**: shows authorized shops grouped by platform, supports multi-select.
2. **Common configuration**: title / description / price / images / package / inventory strategy (the interface reserves `commonConfig`).
3. **Per-target overrides**: overridden within each platform tab (Douyin category attributes, image sync, etc.); **per-target overrides take precedence**.

### Single-Product API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/products/:id/publish-targets` | Publishable platforms and shops with capability tiers |
| POST | `/api/v1/products/:id/publish-targets/check` | Independent precheck for multiple targets (no DB writes, no platform write API calls) |
| POST | `/api/v1/products/:id/publish-targets/create-drafts` | Batch-creates publish drafts; one subtask per target, aggregated into `product_publish_batches` (`batch_type=single_product`) |

## Multi-Product Bulk Publishing (Phase A2)

### Scenarios

```text
Multiple products → single platform, single shop
Multiple products → single platform, multiple shops
Multiple products → multiple platforms, multiple shops
```

### Operator Entry Points

- Product draft list: multi-select → **Bulk create publish drafts**
- Wizard page: `/product/publish-batch?productIds=...` (5 steps)
- Batch list: Products → Publish tasks → **Publish Batches** tab
- Batch detail: `/product/publish-batches/:id`

### Bulk API

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/product-publish/targets` | Global platform / shop capabilities (wizard step 2) |
| POST | `/api/v1/product-publish/batch-targets/check` | Precheck for the multi-product × multi-target matrix |
| POST | `/api/v1/product-publish/batch-targets/create-drafts` | Creates a multi-product batch with subtasks |
| GET | `/api/v1/product-publish/batches` | Batch list |
| GET | `/api/v1/product-publish/batches/:id` | Batch detail + subtasks |
| POST | `/api/v1/product-publish/batches/:id/retry-failed` | Retries only the failed subtasks |
| POST | `/api/v1/product-publish/batches/:id/cancel-pending` | Cancels only the pending subtasks |

### Check-Response Summary

- `ready` → draft can be created
- `warning` → recommended to review (can proceed, requires manual confirmation)
- `blocked` → draft cannot be created yet

Each **product × target** combination's `issues[]` includes a localized `title` / `message`; the internal code is in `technicalDetails.rawCode`.

Creation parameters:

- `onlyReady=true`: creates only `ready` items
- `includeWarnings=false`: skips `warning` items
- `blocked` items are skipped by default and cannot be force-submitted

### Batch and Subtask Model

- `product_publish_batches`: when `batch_type=multi_product`, `product_id` can be null; stores `product_count`, `target_count`, `task_count`, and a configuration snapshot `input`
- `product_publish_tasks.batch_id` + `target_key`: one subtask per product × per target
- Subtask `input` stores a snapshot of `effectiveConfig` + `configSources`

Batch status: `pending` / `running` / `partial_success` / `success` / `failed` / `cancelled`

### Configuration Priority

```text
System default → Platform default → Shop default → Bulk common configuration → Product override → Platform override → Shop override → Product+platform+shop override
```

MVP common fields: `priceRule`, `imageStrategy`, `stockStrategy`, `packageWeight`, `packageSize`, `remark`

### Idempotency

- Batch: `publish-batch:{userId}:{productIdsHash}:{targetsHash}:{configHash}`
- Subtask: not recreated when the same product + shop + platform already has a successful Douyin / local draft

### Operation Logs

- `product.publish.batch.check`
- `product.publish.batch.create`
- `product.publish.batch.retry_failed`
- `product.publish.batch.cancel_pending`

Failed task center: when a subtask has a `batch_id`, its detail link navigates to the batch detail page.

## Boundary with Direct Listing

- This phase is accurately named **bulk creation of publish drafts**, not "one-click publish / direct listing."
- Platforms without a real Provider integration **must not** be disguised as having published successfully.
- Douyin remains a **Release Candidate**; the OpenAPI fields are unchanged; it reuses the existing `create-draft` pipeline.

## Phase A2 Implementation Boundaries (Prohibited)

- Automatic direct listing
- Adding a new real platform OpenAPI
- Modifying Douyin OpenAPI fields
- A single subtask failure rolling back the entire batch
- Disguising `local_draft_only` as a successful real-platform publish

## Next Phase (Not in A2.2)

- Fold title / description strategy into the common configuration (optional before Phase A3)
- Upgrade real `ProductPublishProvider` draft creation for each cross-border platform
- Make batch processing async/queued (currently subtasks are created synchronously; the Douyin async worker remains unchanged)

## Phase A2.1 Acceptance and Production Safety Closure

### Bulk Size Limits

| Environment variable | Default | Description |
| --- | --- | --- |
| `PUBLISH_BATCH_MAX_PRODUCTS` | 100 | Maximum products per batch |
| `PUBLISH_BATCH_MAX_TARGETS` | 20 | Maximum publish targets per batch |
| `PUBLISH_BATCH_MAX_TASKS` | 300 | Cap on products × targets |

Over the limit, HTTP 400: "The selected products and publish targets are too many; please create publish drafts in smaller batches."

### Database Migration

Explicit Postgres migration: [`PUBLISH_BATCH_MIGRATION.md`](PUBLISH_BATCH_MIGRATION.md) (nullable `product_id`, query indexes, partial unique index on `idempotency_key` for active batches).

### Execution Strategy (Conclusion for This Phase)

- **Keep** create-drafts as synchronous orchestration; ≤300 subtasks per batch.
- `local_draft_only`: synchronous DB writes, expected to be acceptable.
- Douyin: subtasks are pending + processed by a Redis worker; production should keep `PRODUCT_PUBLISH_QUEUE_ENABLED=true`.
- No dedicated batch worker queue has been introduced.

### Tests and Scripts

- Integration test: `backend/internal/modules/productpublish/batch_targets_integration_test.go`
- Performance regression runs via GitHub Actions; product experience is manually accepted by maintainers; one-off performance reports are not retained.
- UX and business flows are manually signed off per [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md).

## Phase A2.2 Common Configuration and Override Configuration UI (2026-06-19)

### Current-State Audit (Before the Change)

| Item | Before | After |
| --- | --- | --- |
| Step 3 common configuration | MVP flat fields (`priceRule` as text, etc.) | Structured form: price / images / inventory / package / remark |
| Step 4 per-target overrides | `window.prompt` + hardcoded defaults | Tab table + modal editor, supports add/remove/edit/copy |
| Effective configuration preview | None | Step 5 "View effective configuration" |
| Configuration validation | Matrix size only | Frontend and backend numeric / strategy / privilege validation |
| Wizard state | Easily lost when switching steps | `localStorage` draft (keyed by user + productIds hash) |

### Common Configuration Fields (`commonConfig`)

Nested structure (compatible with A2's legacy flat fields):

```json
{
  "price": { "strategy", "markupValue", "minProfitMargin", "decimalHandling" },
  "image": { "mainImageStrategy", "detailImageStrategy", "preferProcessedImages", "skipFailedImages" },
  "inventory": { "strategy", "safetyStock", "fixedQuantity", "outOfStockAction" },
  "package": { "weight", "weightUnit", "length", "width", "height", "sizeUnit" },
  "remark": "Internal remark"
}
```

### Override Layers (`overrides`)

```text
products[productId]
platforms[platform]
shops[shopId]
productTargets[productId:platform:shopId]
```

### Configuration Priority (Operator-Visible)

```text
System default → Platform default → Shop default → Common configuration → Product override → Platform override → Shop override → Product-target override
```

Subtask `input` still stores `effectiveConfig` + `configSources` (leaf paths, e.g. `price.markupValue`). `retry-failed` recomputes configuration from the batch's `input`; it **does not change** the snapshot of subtasks that already succeeded.

### Configuration Validation Errors

HTTP 400, `code=40004`, `data`:

```json
{
  "code": "PUBLISH_CONFIG_INVALID",
  "title": "Invalid publish configuration",
  "message": "Common inventory must be an integer greater than or equal to 0.",
  "technicalDetails": { "field": "commonConfig.inventory.fixedQuantity" }
}
```

### Frontend Components

- `admin/src/pages/Product/PublishBatch/components/PublishConfigEditor.tsx`
- `admin/src/pages/Product/PublishBatch/components/OverrideConfigTabs.tsx`
- `admin/src/pages/Product/PublishBatch/components/EffectiveConfigPreviewModal.tsx`
- `admin/src/constants/publishConfig.ts`
- `admin/src/utils/publishConfigMerge.ts`

### `window.prompt` Cleanup

**Zero** occurrences of `window.prompt` in the bulk publishing wizard (verified by a project-wide scan of `admin`).

### Still Out of Scope

- Automatic direct listing
- Adding new real cross-border platform OpenAPIs
- Modifying Douyin OpenAPI fields
- Making batch processing async/queued
- Phase A3 bulk AI

### Next Phase

- Fold title / description strategy into the common configuration
- Upgrade real `ProductPublishProvider` draft creation for each cross-border platform
- Make batch processing async/queued (optional)
