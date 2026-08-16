# Unified Idempotency Design (P2 / P2.1)

> Phase P2 introduces cross-module idempotency infrastructure to avoid side effects from duplicate execution. Implemented in `backend/internal/modules/idempotency`.
> Key production write paths are integrated through the shared `idempotency.Service` (order sync/import, inventory deduction/push, publishing, customer-service outbound messages, AI batches, webhooks); current coverage is authoritative in the code, CI regression tests, and [`DOMAIN_IDEMPOTENCY_INTEGRATION.md`](DOMAIN_IDEMPOTENCY_INTEGRATION.md).
> **Phase P2.2**: AI copy/image **apply + undo** keys and webhook **HTTP receive / process** keys are now integrated; see [`AI_RESULT_APPLY_IDEMPOTENCY.md`](AI_RESULT_APPLY_IDEMPOTENCY.md), [`AI_RESULT_UNDO_DESIGN.md`](AI_RESULT_UNDO_DESIGN.md), [`WEBHOOK_HTTP_RECEIVER_DESIGN.md`](WEBHOOK_HTTP_RECEIVER_DESIGN.md).

## Data Model: `idempotency_records`

| Field | Description |
| --- | --- |
| `scope` + `idempotency_key` | Business domain + stable key, **jointly unique** (`ux_idempotency_scope_key`) |
| `request_hash` | SHA-256 of the request body; the same key with a different payload is treated as a conflict |
| `status` | See the state machine below |
| `owner` | Current lock holder (worker ID or service name) |
| `locked_until` | Lease expiration while processing |
| `expires_at` | Retention period for succeeded records (default 7 days, for replay lookups) |
| `resource_type` / `resource_id` | Associated resource after success |
| `response_code` / `response_summary` | Success summary, used for API replay |
| `error_code` / `retryable` | Failure classification |

### State Machine

```text
pending → processing → succeeded
                    ↘ failed_retryable → processing (retry)
                    ↘ failed_permanent
processing (lease expired) → expired (swept by ReleaseExpired)
```

Default lease is **2 minutes** (`DefaultLease`); completed-record TTL is **7 days** (`DefaultTTL`).

## Service API

| Method | Purpose |
| --- | --- |
| `Acquire(ctx, scope, key, requestHash, owner, lease)` | Acquire execution rights; if already succeeded, returns `Replay` + `OpError(IDEMPOTENCY_ALREADY_SUCCEEDED)` |
| `Heartbeat(ctx, recordID, owner, lease)` | Extend the processing lease |
| `Complete(ctx, recordID, owner, CompleteResult)` | Mark as succeeded and release the lease |
| `Fail(ctx, recordID, owner, errorCode, retryable)` | Mark as retryable or permanently failed |
| `Get(ctx, scope, key)` | Look up the latest record |
| `ReleaseExpired(ctx, limit)` | Mark expired processing / TTL-exceeded records as `expired` |

## P2.1 Integration Status

| Status | Description |
| --- | --- |
| **Integrated** | Order sync tasks, order import, inventory deduction/push, publish batch/enqueue, customer-service outbound messages, AI copy/image batch creation, AI copy/image **apply/undo**, webhook inbound ACK + `webhook-process` async handling |
| **Reserved** | Inventory compensation (`inventory-compensate`) |
| **Verification** | Existing Go unit/integration tests, API contract tests, and GitHub Actions; final business-flow verification is manual acceptance |

`router.go` injects the same `idempotencySvc` into `ordersync`, `order`, `inventory`, `productpublish`, `customerchat`, `aiproducttext`, and `aiproductimage`.

## Scope and Key Patterns

Key construction is defined in `scope.go` + `keys.go`; **must not embed secrets or PII**:

| Scope | Key Pattern | Scenario | P2.1 |
| --- | --- | --- | --- |
| `order_sync` | `order-sync-job:{platform}:{shopId}:{mode}:{window}` | Sync task creation | ✓ |
| `order_import` | `order-import:{platform}:{shopId}:{platformOrderId}` | Single order import | ✓ |
| `inventory` | `inventory-deduct:{orderId}:{orderItemId}:{skuId}` | Inventory deduction | ✓ |
| `inventory_push` | `inventory-push:{platform}:{shopId}:{skuId}:{stockVersion}` | Inventory push | ✓ |
| `publish` | `publish-batch:…` / `publish-enqueue:…` | Publish batch/enqueue | ✓ |
| `customer_send` | `customer-send:{conversationId}:{clientMessageId}` | Customer-service outbound messages | ✓ |
| `ai_text` | `ai-text-batch:…` / `ai-text-apply:…` / `ai-text-undo:…` | AI copy batch / apply / undo | ✓ (P2.2 apply/undo) |
| `ai_image` | `ai-image-batch:…` / `ai-image-apply:…` / `ai-image-undo:…` | AI image batch / apply / undo | ✓ (P2.2 apply/undo) |
| `webhook` | `webhook:{platform}:{eventId}` / `webhook-process:…` | Webhook inbound / async processing | ✓ (P2.2 HTTP) |

`HashRequest(payload []byte)` computes SHA-256 over the canonicalized request body.

## Error Codes

| Code | Meaning | Suggested Handling |
| --- | --- | --- |
| `IDEMPOTENCY_IN_PROGRESS` | Another worker holds the lock and is processing | Poll or return 409 |
| `IDEMPOTENCY_KEY_CONFLICT` | Same key with a different payload, or a permanent failure | Requires manual intervention |
| `IDEMPOTENCY_ALREADY_SUCCEEDED` | Already succeeded; `response_summary` can be replayed | Return the cached result |
| `IDEMPOTENCY_LEASE_LOST` | Lease lost (Complete/Fail/Heartbeat) | Abandon this write attempt |
| `IDEMPOTENCY_RECORD_EXPIRED` | Record has expired | Use a new key or retry after cleanup |

## Indexes and Migrations

The reliability migration (`migrate_reliability.go`) creates the table and indexes: `ix_idempotency_status`, `ix_idempotency_locked_until`.

## Usage Conventions

1. Call `Acquire` before any write operation; call `Heartbeat` periodically for long-running tasks.
2. Business success must call `Complete`; failures determine `retryable` via `taskretry.Classify`.
3. Clients may optionally pass an idempotency key; the server must generate the key from stable business semantics, never a random UUID.
4. Webhooks and order sync share the same `idempotency_records` plus dual-write duplicate protection with the domain tables.
5. Async workers must work together with `tasklease` (`execution_id` / `heartbeat_at` / `lock_version`); see [`TASK_LEASE_AND_HEARTBEAT_DESIGN.md`](TASK_LEASE_AND_HEARTBEAT_DESIGN.md).

## P3.2 Douyin Webhook Scoped Keys

P3.2 Douyin webhook uses scoped keys instead of the historical P2.2 shape:

```text
webhook:{platform}:{tenantId}:{platformShopId}:{eventId}
webhook-process:{platform}:{tenantId}:{platformShopId}:{eventId}
```

Do not use an app secret, access token, refresh token, buyer data, or raw payload as any part of these keys.
