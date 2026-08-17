# Stale Worker Protection (P2.1)

> When a worker process hangs, a network partition occurs, or a GC pause happens, a **stale execution** must be prevented from continuing to write task results or trigger duplicate side effects. P2.1 implements this via a three-layer mechanism: **lease TTL + heartbeat + execution_id**.

## Threat Model

| Scenario | Consequence without protection | P2.1 protection |
| --- | --- | --- |
| Worker A stops-the-world for a long time after claiming | A still writes successfully on recovery, duplicating B's execution | `locked_until` expiry + `ValidateLease` rejection |
| Worker A and B both believe they hold the task | Duplicate writes to platform/inventory | Atomic claim condition + unique valid `execution_id` |
| Heartbeat stops but the process hasn't exited | A fake "running" state occupies the slot | `heartbeat_at` + Reaper / `TakeoverExpired` |
| Duplicate queue delivery | Duplicate business side effects | `idempotency.Service` business keys |

## Mechanism Details

### 1. Lease TTL (`locked_until`)

- On claim, set `locked_until = now + leaseTTL` (default roughly 90s-180s, configurable per module).
- If renewal fails or stops, the execution **becomes invalid** once the TTL expires.

### 2. Heartbeat (`heartbeat_at`)

- `tasklease.StartRenewal` calls `RenewHeartbeat` every `TTL/3`.
- Updates `heartbeat_at` and `locked_until`.
- `TakeoverExpired` requires `heartbeat_at < staleCutoff`, avoiding mistakenly seizing a task from a still-healthy worker.

### 3. Execution Identity (`execution_id` + `lock_version`)

Each claim / takeover generates a new UUID and increments `lock_version`.

The SQL that writes results must include:

```sql
WHERE id = ? AND locked_by = ? AND execution_id = ? AND lock_version = ?
```

See each module's `finish*Task` implementation (e.g. `ordersync/lease.go`).

### 4. Reaper and Legacy Reclamation

`WORKER_REAPER_ENABLED` scans for:

- `status = running AND locked_until < now` → marked `failed` / `retrying` / `lease_expired`.
- No lease metadata and `updated_at` older than `WORKER_LEGACY_RUNNING_TIMEOUT_SECONDS` → legacy reclamation (publishing module `RecoverLegacyRunning`).

### 5. Business Idempotency Fallback

Even if a duplicate execution occurs after a task lease has expired, critical write paths are still protected by `idempotency_records` (see [`DOMAIN_IDEMPOTENCY_INTEGRATION.md`](DOMAIN_IDEMPOTENCY_INTEGRATION.md)).

## Worker-Side Conventions

1. After a successful claim, `StartRenewal` **must** be called, with `defer stop()`.
2. Before calling a third-party API, `ValidateLease` may be called (optional before a long HTTP call).
3. Before writing a final state to the DB, a conditional update **must** be used; on `ErrLeaseLost`:
   - Do not idempotently call Complete/Fail (if a record isn't already held) — or handle idempotently based on the record's state;
   - Log the event and rely on queue redelivery or manual retry.
4. Writing `succeeded` back to the task table after a lease is lost is **prohibited**.

## Operational Signals

| Signal | Meaning |
| --- | --- |
| Failed task `lease_expired` | Worker did not renew the lease in time, or the process died |
| Idempotency `IDEMPOTENCY_LEASE_LOST` | Business orchestration held the lock past its timeout |
| `/health` workers block | Whether process-level heartbeats are healthy |
| `ix_*_heartbeat_at` index | Supports Reaper scans |

The configuration status center's **task row-level heartbeat and lease** item (`configstatus/domain_idempotency_status.go`) is marked as configured.

## Boundary with "Production Ready"

Stale worker protection reduces the risk of duplicate writes; it **does not** equate to passing production acceptance. AI provider keys, public Storage access, real Doudian E2E tests, and other items may still block final acceptance (see `PROGRESS.md`).

## Related Documents

- [`TASK_LEASE_AND_HEARTBEAT_DESIGN.md`](TASK_LEASE_AND_HEARTBEAT_DESIGN.md)
- [`CONCURRENT_WRITE_SAFETY.md`](CONCURRENT_WRITE_SAFETY.md)
- [`TASK_RELIABILITY_DESIGN.md`](TASK_RELIABILITY_DESIGN.md)
