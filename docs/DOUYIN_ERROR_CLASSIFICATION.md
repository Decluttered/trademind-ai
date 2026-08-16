# Douyin Shop Error Classification Design

## ErrorClass Enum

| ErrorClass | Meaning | Typical Error Codes | SafeRetry | ManualReview |
|-----------|------|-----------|-----------|--------------|
| `auth_error` | Authentication/token invalid | DOUYIN_AUTH_EXPIRED, DOUYIN_REAUTHORIZATION_REQUIRED | false | false |
| `rate_limited` | Rate limiting | DOUYIN_RATE_LIMITED | true | false |
| `timeout` | Timeout (read side) | DOUYIN_REQUEST_TIMEOUT | true | false |
| `unknown_result` | Write timed out, result unknown | DOUYIN_UNKNOWN_RESULT | false | **true** |
| `contract_mismatch` | API shape mismatch / contract not verified | DOUYIN_CONTRACT_MISMATCH | false | false |
| `validation` | Request parameter validation failed | DOUYIN_VALIDATION_FAILED | false | false |
| `permission` | Permission denied | DOUYIN_PERMISSION_DENIED | false | false |
| `not_found` | Resource does not exist | DOUYIN_RESOURCE_NOT_FOUND, DOUYIN_PRODUCT_NOT_FOUND | false | false |
| `network` | Network-layer error | — | true | false |
| `system` | System/configuration error | DOUYIN_NOT_CONFIGURED | false | false |

## unknown_result Handling Rules

After a write operation (product.addV2, sku.syncStock, image upload) times out:

1. `UnknownResult = true`
2. `SafeRetry = false` (automatic retry forbidden)
3. `ManualReviewRequired = true` (requires manual review)
4. The task status is marked `unknown_result`
5. Before retrying, call `tryRecoverDouyinDraftFromPlatform` to confirm the operation's actual result

## ClassifyError Function

```go
errClass := douyinshop.ClassifyError(err)
```

Prefers the `Error.ErrorClass` field, falls back to flag-based inference (AuthExpired, RateLimited, PermissionDenied), and finally falls back to string matching.

## Log Safety

The following fields must never appear in logs:
- `access_token`
- `refresh_token`
- `app_secret`
- `password`
- `cookie`

`SanitizeErrorText` filters sensitive terms via `safefields.RedactString`.
