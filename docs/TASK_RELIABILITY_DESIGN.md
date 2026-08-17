# Task Reliability Design (P2 / P2.2)

> Async tasks (collection, images, order sync, customer service sync, publishing, inventory sync) uniformly adopt a **DB lease + heartbeat renewal + retry policy + dead letter** pattern.
> All six workers above are integrated with the shared `tasklease` package (`TryClaim` / `TryClaimPendingOrRetrying`, `execution_id`, heartbeat renewal, `ValidateLease` + `finish*Task` guarded write-back). Current behavior is defined by the code and CI regression; the historical matrix can be traced from Git history.

## Task Lease Fields

Common across all task tables:

| Field | Description |
| --- | --- |
| `locked_by` | Worker instance ID (`worker.GenerateWorkerID` / Registry) |
| `locked_until` | Lease expiration time (UTC) |
| `lock_version` | Optimistic lock version, incremented `+1` on claim |

Claim condition: `status` is pending/retrying (and `next_retry_at` has elapsed), and `locked_by IS NULL OR locked_until < now`.

## Heartbeat and Renewal

- The lease TTL comes from each queue's env var (e.g. `COLLECT_TASK_TIMEOUT_SECONDS`, `ORDER_SYNC_TASK_TIMEOUT_SECONDS`).
- The collection lease includes a margin of ≥ `COLLECTOR_TIMEOUT_SECONDS + 60s`.
- A background goroutine refreshes `locked_until` every **TTL/3** (minimum 5s).
- The worker process writes `worker_instances.last_heartbeat_at` via `worker.Registry` (`WORKER_HEARTBEAT_ENABLED`).

## Reaper (Expiry Reclamation)

`WORKER_REAPER_ENABLED` periodically scans for:

- `locked_until < now` while still `running` → marked failed/retrying, or a `lease_expired` event.
- `WORKER_LEGACY_RUNNING_TIMEOUT_SECONDS` cleans up historical stuck rows that have no lease metadata.

The configuration status center's **task lease** item is marked ready.

## Retry Policy (`taskretry`)

Default `Policy`:

| Attempt | Backoff |
| --- | --- |
| 1 | Immediate |
| 2 | 30s |
| 3 | 2m |
| 4 | 10m |
| 5 | 30m |

- `MaxAttempts = 5`, `JitterRatio = 0.15`, `MaxDelay = 30m`.
- `ShouldRetry(attempt, retryable)` controls whether to continue.
- HTTP 429 / 5xx / timeout / network error → `retryable=true`.
- Permission, validation, and idempotency conflicts → `retryable=false`.

## Error Classification (excerpt)

| Code | Retryable |
| --- | --- |
| `timeout`, `network_error`, `provider_5xx`, `rate_limited` | Yes |
| `lease_expired`, `redis_temporary_failure` | Yes |
| `permission_denied`, `validation_failed`, `idempotency_conflict` | No |
| `credential_refresh_required` | No |

## Dead Letter (`dead_letter`)

- Collection tasks define `StatusDeadLetter = "dead_letter"`.
- `taskretry.Policy.IsDeadLetter(attempt)`: once attempt ≥ MaxAttempts, the task enters dead-letter semantics.
- Failed task center: when `DeadLetter=true`, `SafeRetry=false`, requiring manual handling or a new task created with adjusted parameters.
- Dead-lettered tasks retain the failure reason and `retry_count`, and are not automatically re-queued.

## Cooperation with the Idempotency Service

Long-running orchestration can use `idempotency.Service` in parallel:

1. `Acquire` obtains business-level execution rights;
2. The task lease guarantees single-worker consumption;
3. On success, `Complete` the idempotency record; on failure, `Fail(retryable)`.

## Operations

- Env: `WORKER_HEARTBEAT_*`, `WORKER_REAPER_*`, `COLLECT_TASK_TIMEOUT_SECONDS`, etc. — see `.env.example`.
- `/health` reports queue depth; the failed task center can filter by `lease_expired` / `rate_limited`.
- Multiple instances require a Redis queue plus a DB lease.
