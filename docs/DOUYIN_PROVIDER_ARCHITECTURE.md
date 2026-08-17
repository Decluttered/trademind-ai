# Douyin Shop Provider Architecture Design

> Phase: P3 | Status: Code implemented, pending real-credential E2E verification

## Layering

```
shop/douyin_oauth.go          ← OAuth authorization entry point, token persistence
    ↓
douyinshop/client.go          ← HTTP client + token refresh + singleflight
    ↓
douyinshop/facade.go          ← DouyinProvider interface (unified entry point for all capabilities)
    ↓
douyinshop/{category,image,product,order,inventory,customer,brand}.go
                               ← Concrete implementations of each capability
    ↓
douyinshop/sign.go            ← Request signing (App Key + timestamp + parameter MD5)
douyinshop/request.go         ← HTTP request construction
douyinshop/response.go        ← Response parsing, platform error mapping
douyinshop/errors.go          ← Unified error types + ErrorClass
```

## Key Design Decisions

### 1. HTTPDoer Interface
The client injects its HTTP implementation via the `HTTPDoer` interface, making it easy to mock in tests. Defaults to `*http.Client`.

### 2. Token Refresh Protection
- `EnsureFreshAccessSingleflight`: only one token refresh request per shop at a time
- `TokenVersion` (added in P3): prevents an older-version token from overwriting a newer one in multi-instance scenarios

### 3. Error Classification
All errors propagate through the `*Error` type, which includes:
- `ErrorClass`: `auth_error`, `rate_limited`, `timeout`, `unknown_result`, `contract_mismatch`, etc.
- `UnknownResult`: set to true after a write times out, disabling automatic retries
- `SafeRetry`: true when idempotency-safe (read-only operations)
- `RetryAfter`: taken from the Retry-After response header

### 4. Write Idempotency
- Product drafts: `douyin-product-draft-create:{shopId}:{draftId}:{version}`
- Image uploads: `douyin-image-upload:{shopId}:{objectKey}:{contentHash}`
- After a timeout, `tryRecoverDouyinDraftFromPlatform` runs first before deciding whether to retry

### 5. External Dependency Constraints
- Customer service messaging API: `blocked_by_contract_verification`
- Brand list API: `blocked_by_contract_verification`
- All OpenAPI calls: require a real App Key and an authorized shop token
