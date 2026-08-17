# Domain Idempotency Integration Guide (P2.1)

> Guides new business write paths on integrating with the unified `backend/internal/modules/idempotency` service. P2.1 has completed integration of the key paths; this document also serves as a template for future extensions.

## When Integration Is Required

The following operations **must** go through idempotency:

- Write operations that produce **side effects that are not easily reversible** against a third party or the DB (deducting inventory, sending messages, creating platform orders/products, webhook processing).
- Operations where the client or queue may **retry** the same business request (network timeout, 429, worker redelivery).
- Operations where multiple API/worker instances may process the same business key **concurrently**.

Read-only queries, pure in-memory computation, and paths that already have a DB unique constraint and don't need to replay a response digest can be evaluated for exemption (must be explained in the PR).

## Integration Steps

### 1. Define Scope and Key

Add a business scope constant in `idempotency/scope.go` (if it doesn't already exist):

```go
const ScopeMyFeature = "my_feature"
```

Add a stable key constructor in `idempotency/keys.go`. **Do not** embed API keys, tokens, or buyer PII:

```go
func MyFeatureAction(shopID, businessID string) string {
    return fmt.Sprintf("my-feature:%s:%s", norm(shopID), norm(businessID))
}
```

The key must consist only of business-semantic fields (platform, shop, order number, client messageId, content version hash, etc.).

### 2. Inject the Service

Inject the shared instance into the module's service in `router.go`:

```go
idempotencySvc := &idempotency.Service{DB: dep.DB}
mySvc := &myfeature.Service{DB: dep.DB, Idempotency: idempotencySvc}
```

### 3. Acquire → Execute Business Logic → Complete / Fail

Recommended pattern (consistent with the P2.1 modules):

```go
owner := idempotency.OwnerFromRequest(c.GetString("requestId"), "my-feature")
key := idempotency.MyFeatureAction(shopID, bizID)
hash := idempotency.HashRequest(payloadBytes)

res, err := s.Idempotency.Acquire(ctx, idempotency.ScopeMyFeature, key, hash, owner, idempotency.DefaultLease)
decision, rec, _ := idempotency.Classify(res, err)
switch decision {
case idempotency.DecisionAlreadySucceeded:
    return replayCachedResult(res)
case idempotency.DecisionInProgress:
    return conflictInProgress()
case idempotency.DecisionKeyConflict, idempotency.DecisionPermanentFailure:
    return permanentConflict()
case idempotency.DecisionAcquired, idempotency.DecisionRetryAllowed:
    // fall through
}

// Long-running tasks can call Heartbeat(ctx, rec.ID, owner, lease)
if bizErr != nil {
    retryable := taskretry.Classify(bizErr).Retryable
    _ = s.Idempotency.Fail(ctx, rec.ID, owner, bizErr.Error(), retryable)
    return bizErr
}
return s.Idempotency.Complete(ctx, rec.ID, owner, idempotency.CompleteResult{
    ResponseCode:    "MY_FEATURE_OK",
    ResponseSummary: summaryJSON,
    ResourceType:    "my_resource",
    ResourceID:      resourceID,
})
```

`Service.Execute` can also be used to simplify Acquire+Complete orchestration (see `execute.go`).

### 4. Dual-Write with Domain Tables

The idempotency record stores **execution ownership and a replay digest**; the business fact is still written to the domain table, and a DB-level unique constraint is retained as the last line of defense, for example:

| Path | Domain-level duplicate prevention |
| --- | --- |
| Order import | Unique index on platform order |
| Inventory deduction | `inventory_change_logs.business_event_key` |
| Webhook | `webhook_events(platform, event_id)` |
| Listing batch | `product_publish_batches.idempotency_key` |

### 5. HTTP / API Conventions

- The client **may optionally** pass `Idempotency-Key` or a JSON `idempotencyKey`; the server **must** be able to generate a stable key on its own.
- Already succeeded: `IDEMPOTENCY_ALREADY_SUCCEEDED` → return 200 + cached digest or domain resource ID.
- In progress: `IDEMPOTENCY_IN_PROGRESS` → 409 or 202 + polling guidance.
- Key conflict (different payload): `IDEMPOTENCY_KEY_CONFLICT` → 409, requires manual intervention.

### 6. Coordinating with Task Leases

Async worker paths should use both:

1. **Business idempotency** (`idempotency.Service`) — prevents duplicate creation of side effects;
2. **Task lease** (`tasklease`) — ensures only one execution writes the result for a given `task_id`.

Long-running tasks call `tasklease.StartRenewal` after claiming; before submitting the result, use `ValidateLease` or `finish*Task` with a conditional update on `execution_id` + `lock_version`.

### 7. Testing and Scanning

- Unit/integration tests should cover: concurrent Acquire, replay of an already-succeeded result, payload conflicts, and lease loss.
- Run before submitting:

```bash
go test ./internal/modules/idempotency/... ./internal/pkg/tasklease/...
```

Full automated regression is executed by GitHub Actions; there is no separately maintained staged static gate.

## Index of Paths Integrated in P2.1

| Module | File |
| --- | --- |
| Order sync | `ordersync/idempotency_create.go` |
| Order import | `order/idempotency_import.go` |
| Inventory deduction | `inventory/idempotency_deduct.go` |
| Inventory push | `inventory/idempotency_push.go` |
| Listing | `productpublish/idempotency_batch.go` |
| Outbound customer service messages | `customerchat/send_platform.go` |
| AI copy batch | `aiproducttext/service.go` (`acquireTextBatch`) |
| AI image batch | `aiproductimage/service.go` (`acquireImageBatch`) |
| Webhook | `webhook/service.go` |

The current integration scope is authoritative in the code and GitHub Actions regression; the historical matrix can be traced through Git history.

## Common Errors

| Symptom | Cause | Handling |
| --- | --- | --- |
| Duplicate inventory deduction | Acquire not called, or key is unstable | Use a fixed key based on orderItem+sku |
| Complete returns lease lost | Worker lease expired or was taken over | Abandon the write; rely on idempotent replay |
| Same key, different body → 409 | Client reused the key but changed the payload | Use a new key or wait for TTL expiry |
| DB unique constraint only, no idempotency | Retry returns 500 instead of replaying | Add Acquire + Complete digest |
## P3.2 Douyin Webhook Key Update

The P2.1 `webhook_events(platform, event_id)` wording is historical for the foundation receiver. For Douyin webhook business handling, the domain key is now `webhook_events(platform, tenant_id, platform_shop_id, event_id)`, with idempotency keys generated by `WebhookScoped` / `WebhookProcessScoped`.

Keep future platform adapters from falling back to a global or implicit single-shop event key.
