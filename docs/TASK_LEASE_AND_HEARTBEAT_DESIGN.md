# Task Lease and Heartbeat Design (P2.1)

> P2.1 introduces the shared package `backend/internal/pkg/tasklease`, which adds an **execution identity** and a **heartbeat timestamp** on top of the existing P2 `locked_by` / `locked_until` / `lock_version` fields, to prevent stale writes from multiple workers.

## Design Goals

1. **One execution per task**: at most one valid execution for a given `task_id` while in the `running` state.
2. **Detectable lease loss**: before writing results, a worker must verify it still holds the current `execution_id` + `lock_version`.
3. **Recoverable**: after a lease or heartbeat expires, another worker can take over (Takeover), or the Reaper marks the task as failed/retrying.
4. **Orthogonal to business idempotency**: the task lease governs **queue consumption**; `idempotency.Service` governs **business side effects**.

## Data Fields

Each async task table (from P2.1 onward) is extended with:

| Field | Type | Description |
| --- | --- | --- |
| `locked_by` | string | Worker instance ID |
| `locked_until` | timestamptz | Lease expiration time |
| `lock_version` | int | Incremented by **+1** on each claim/takeover; optimistic lock |
| `heartbeat_at` | timestamptz | Timestamp of the most recent renewal |
| `execution_id` | uuid (varchar36) | Execution identity generated on this claim |

Migration: `database/migrate_task_execution_tracking.go` (PostgreSQL `ADD COLUMN IF NOT EXISTS` + indexes).

## API (`tasklease` package)

| Function | Purpose |
| --- | --- |
| `TryClaim` | Atomically claims a task from `pending` to `running`, assigns a new `execution_id`, increments `lock_version`, and writes `heartbeat_at` |
| `RenewHeartbeat` | Extends `locked_until` and refreshes `heartbeat_at`, keyed by `execution_id` + `lock_version` |
| `ValidateLease` | Verifies a valid lease is still held before writing results |
| `StartRenewal` | A background goroutine that calls `RenewHeartbeat` every `TTL/3` (minimum 5s) |
| `TakeoverExpired` | Lets a new worker take over once both the lease and heartbeat have expired |

Claim condition (simplified):

```text
status = pending AND (locked_by IS NULL OR locked_until < now)
```

Renew / Validate condition:

```text
status = running
AND locked_by = worker
AND execution_id = ?
AND lock_version = ?
AND locked_until >= now
```

## Module Integration (P2.1)

| Module | File | Default TTL source |
| --- | --- | --- |
| Order sync | `ordersync/lease.go` | `ORDER_SYNC_TASK_TIMEOUT_SECONDS` |
| Inventory sync | `inventory/lease.go` | `INVENTORY_SYNC_TASK_TIMEOUT_SECONDS` |
| Product publishing | `productpublish/lease.go` | `PRODUCT_PUBLISH_TASK_TIMEOUT_SECONDS` |

Typical worker flow:

```text
BRPOP queue
  → TryClaim(taskID)
  → StartRenewal(...)   // defer stop()
  → execute business logic (may run idempotency.Acquire in parallel)
  → ValidateLease / finish*Task(updates)  // WHERE execution_id + lock_version
  → update status=succeeded|failed
```

`finish*Task` uses a conditional update; when `RowsAffected == 0`, it returns `tasklease.ErrLeaseLost`, and the caller **must not** continue writing to the platform or inventory.

## Relationship to the Worker Registry

- **Task row-level**: `heartbeat_at` / `locked_until` — ownership of a single task's execution.
- **Process-level**: `worker_instances.last_heartbeat_at` — lets operations see whether a worker process is alive (see `TASK_RELIABILITY_DESIGN.md`).

The two are complementary: if a process crashes but the task lease hasn't expired, the task is still protected by the DB lease; once the lease expires, the Reaper or a new worker can recover it.

## Configuration

Related env vars are documented in `.env.example`:

- `ORDER_SYNC_TASK_TIMEOUT_SECONDS`
- `INVENTORY_SYNC_TASK_TIMEOUT_SECONDS`
- `PRODUCT_PUBLISH_TASK_TIMEOUT_SECONDS`
- `WORKER_REAPER_ENABLED` / `WORKER_REAPER_INTERVAL_SECONDS`
- `WORKER_LEGACY_RUNNING_TIMEOUT_SECONDS`

## Follow-up Module Migration

`collect_tasks`, `image_tasks`, and `customer_message_sync_tasks` already gained the new columns in the P2.1 migration; their workers can still use the module's legacy lease logic for now, with a later unified migration to the `tasklease` package (see the `partial` row in the integration matrix).

## Related Documents

- [`STALE_WORKER_PROTECTION.md`](STALE_WORKER_PROTECTION.md)
- [`CONCURRENT_WRITE_SAFETY.md`](CONCURRENT_WRITE_SAFETY.md)
- [`MULTI_INSTANCE_SAFETY.md`](MULTI_INSTANCE_SAFETY.md)
- [`TASK_RELIABILITY_DESIGN.md`](TASK_RELIABILITY_DESIGN.md)
