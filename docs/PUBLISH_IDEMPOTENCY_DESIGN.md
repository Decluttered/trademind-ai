# Publishing Idempotency Design (P2)

> Both multi-product batch publishing and single-task create-draft flows must prevent duplicate subtask creation and duplicate platform API calls.

## Two Idempotency Layers

| Layer | Storage | Granularity |
| --- | --- | --- |
| Batch | `product_publish_batches.idempotency_key` | One wizard submission |
| Subtask | `product_publish_tasks.input.idempotencyKey` | Product × Platform × Shop × Effective Config |

Unified idempotency key (optionally integrated with `idempotency_records`):

```text
idempotency.PublishDraft(shopId, productDraftId, publishVersion)
→ publish-draft:{shopId}:{productDraftId}:{publishVersion}
```

## Batch Key Generation

`batchIdempotencyKey(adminId, productIds, targets, commonConfig, overrides)`:

```text
publish-batch:{adminId}:{configHash}
```

- `productIds` and targets (`platform:shopId`) are **sorted before hashing**, so order does not matter.
- `configHash` incorporates `commonConfig` and the four layers of `overrides`.
- Integration tests verify that out-of-order productIds produce the same key.

### Active Batch Unique Index

```sql
CREATE UNIQUE INDEX ux_publish_batches_idempotency_active
 ON product_publish_batches (idempotency_key)
 WHERE idempotency_key <> '' AND status NOT IN ('failed','cancelled');
```

Duplicate `idempotency_key` values are checked before migration; if found, the migration is blocked and requires manual cleanup.

### Duplicate Submission Behavior

1. Before creation, query for `idempotency_key` with status ∉ `{failed, cancelled}`.
2. Hit → return the existing batch's response (`batchCreateResponseFromExisting`).
3. `Create` hits a unique conflict → query again and return the existing batch (race-condition fallback).

## Subtask Key Generation

`taskIdempotencyKey(productId, platform, shopId, effectiveConfig)`:

```text
publish-task:{productId}:{platform}:{shopId}:{configHash(effectiveConfig)}
```

`effectiveConfig` is the deep-merge result of the common config and overrides; a config change produces a new key, allowing a legitimate second publish.

## Successful Task Deduplication

`findExistingSuccessfulTask`: for the same product + platform + shop + effectiveConfig that already succeeded → during batch creation, **reference the existing task** instead of creating a new one or calling the platform API.

`retry-failed` only targets failed/cancelled items; concurrent claims prevent two workers from retrying the same subtask.

## Single-Product create-draft

The "Create Douyin Draft" action on the product detail page goes through the `product_publish_tasks` queue:

- Worker DB lease + `PRODUCT_PUBLISH_TASK_TIMEOUT_SECONDS`.
- Douyin Phase 10.2: `product.detail` looks back up via `out_product_id` to avoid a duplicate `product.addV2` call after a timeout.

## Validation and Limits

- Batch limit env vars: `PUBLISH_BATCH_MAX_PRODUCTS=100`, `MAX_TARGETS=20`, `MAX_TASKS=300`.
- Repeated clicks with the same parameters → same `batchId`; changing overrides → new key; a `failed` batch does not block a new submission.
- See `MULTI_PLATFORM_PUBLISHING_DESIGN.md`, `PUBLISH_BATCH_MIGRATION.md` for details.
