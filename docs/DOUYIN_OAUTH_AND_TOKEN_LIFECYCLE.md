# Douyin Shop OAuth and Token Lifecycle

## OAuth Authorization Flow

```
1. Admin initiates authorization
   POST /api/v1/shops/douyin/oauth/start
   → generate a (random) state
   → Redis SET oauth:douyin_shop:state:{state} = payload (10min TTL)
   → DB INSERT douyin_oauth_states (StateHash, ExpiresAt)
   → return the authorization URL (includes service_id + state)

2. User completes authorization in the browser
   → Douyin Shop calls back GET /api/v1/shops/douyin/oauth/callback?code=xxx&state=yyy

3. Callback handling
   → Redis GET state (fast path)
   → DB ConsumedAt one-time consumption guard
   → ExchangeCode → AccessToken + RefreshToken
   → persist to shop_auth_tokens (stored encrypted)
   → update shops.auth_status = authorized
```

## State Protection

| Mechanism | Implementation |
|---------|------|
| Redis TTL | Expires after 10 minutes |
| DB ExpiresAt | Redundant expiry check |
| ConsumedAt | Single-use, prevents replay |
| redirect_uri allowlist | Must exactly match configuration |

## Token Refresh

```
EnsureFreshAccess()
  → freshAccessToken() — returns directly if still valid
  → EnsureFreshAccessSingleflight() — dedup
    → ensureFreshAccessDirect()
      → refreshUsable() — whether the refresh token is usable
      → RefreshToken API (token.refresh)
      → PersistRefreshedToken callback
      → update the Client's in-memory token
```

## TokenVersion (Added in P3)

`token_version` increments after every successful refresh. The version is validated before writing to `shop_auth_tokens`, preventing concurrent refreshes across multiple instances from overwriting a newer version with an older one (DOUYIN_TOKEN_VERSION_CONFLICT).

## New ShopAuthToken Fields (P3)

| Field | Type | Description |
|-----|------|------|
| token_version | bigint | Monotonically increasing version number |
| reauthorization_required | bool | Set to true when re-authorization is required |
| last_refresh_error_code | varchar(128) | Error code from the most recent refresh failure |

## Error Codes

| Error Code | Meaning |
|-------|------|
| DOUYIN_AUTH_EXPIRED | Token has expired, re-authorization required |
| DOUYIN_TOKEN_REFRESH_FAILED | Refresh failed |
| DOUYIN_TOKEN_VERSION_CONFLICT | Version conflict, rejects writing an older version |
| DOUYIN_TOKEN_REFRESH_IN_PROGRESS | Refresh in progress |
| DOUYIN_REAUTHORIZATION_REQUIRED | User must complete OAuth again |
| DOUYIN_OAUTH_STATE_MISSING | State not found |
| DOUYIN_OAUTH_STATE_EXPIRED | State has expired |
| DOUYIN_OAUTH_STATE_ALREADY_USED | State has already been used |
| DOUYIN_OAUTH_REDIRECT_NOT_ALLOWED | redirect_uri not in the allowlist |
