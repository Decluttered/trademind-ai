# Provider Resilience Design (P2)

> Outbound calls to third-party platforms, collection services, AI, storage, etc. are all protected by **`httpclient` + timeouts + optional circuit breaking + health caching** on top of the core business logic.

## HTTP Client (`internal/pkg/httpclient`)

### Default Configuration `DefaultConfig()`

| Item | Default |
| --- | --- |
| `ConnectTimeout` | 10s |
| `RequestTimeout` | 60s |
| `ResponseHeaderTimeout` | 30s |
| `MaxResponseBytes` | 32 MiB |
| `MaxRedirects` | 3 |
| `RetryPolicy` | `taskretry.DefaultPolicy()` (5 attempts) |
| `UserAgent` | `TradeMind/1.0` |

### Core Methods

- `Do(ctx, req)`: Single request; optional semaphore `maxConcurrent` for rate limiting.
- `DoWithRetry(ctx, build)`: Automatic backoff retry for 5xx / 429 / network errors.
- `ReadLimitedBody(resp)`: Prevents OOM from oversized responses.
- `RedactURL(url)`: Redacts query credentials in logs.

### Circuit Breaker Attachment

```go
client.SetCircuitBreaker(httpclient.NewCircuitBreaker(threshold, openDuration))
```

Default threshold is **5**, open window is **30s** (see `CIRCUIT_BREAKER_AND_RATE_LIMIT.md`).

While the circuit is open, `Do` returns `circuit_open: provider temporarily unavailable` and **does not issue the network request**.

## Error Classification Integration

`taskretry.Classify(err, httpStatus)` drives:

- Whether to retry (`DoWithRetry` / task worker).
- The code shown in the failed task center (`rate_limited`, `provider_5xx`, etc.).
- The idempotency `Fail(retryable)` decision.

## Provider Health (`internal/pkg/providerhealth`)

| State | Meaning |
| --- | --- |
| `available` | Probe succeeded |
| `degraded` | Available but with latency/partial failures |
| `rate_limited` | Currently rate-limited |
| `circuit_open` | Circuit breaker open |
| `unauthorized` | Credential issue |
| `not_configured` | No checker configured |
| `temporary_unavailable` | Briefly unavailable |
| `manual_required` | Requires manual intervention |

- `Registry` cache TTL defaults to **5 minutes**.
- `Get(ctx, provider, capability, force)` re-probes after expiry.
- Config status center **Provider Health** item: cached check, failures do not affect `/health/live`.

## Layering Convention

```text
Handler → Service → Provider interface → httpclient → Third-party API
```

- Provider implementations construct their own `httpclient.Client` and must not bypass the timeout.
- Platform SDKs (COS/OSS) use their own timeout configuration, kept semantically aligned with httpclient.
- Logs must never output a full API key / token; use `RedactURL` and raw truncation.

## Production Checks

Staging/production must not use the local `STORAGE_PROVIDER`; each platform's `timeout_sec` should take the smaller of itself and the httpclient value. See `provider.md`, `CIRCUIT_BREAKER_AND_RATE_LIMIT.md` for details.
