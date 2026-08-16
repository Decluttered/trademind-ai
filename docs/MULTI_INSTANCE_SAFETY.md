# Multi-Instance Safety Design (P2)

> TradeMind API supports horizontal scaling: **migration locks, DB-based task leases, and worker registration heartbeats** prevent double writes and duplicate consumption.

## Architectural Assumptions

```text
        ┌─────────┐   ┌─────────┐
        │ API #1  │   │ API #2  │
        └────┬────┘   └────┬────┘
             │  Redis queue   │
             └────────┬────────┘
                      │
                 PostgreSQL
```

- Stateless HTTP: JWT sessions, not dependent on single-machine in-memory sessions.
- Stateful async: task state lives in DB + Redis LIST; workers can run in any API process.

## Migration Lock (PostgreSQL Advisory Lock)

When multiple instances run `AutoMigrate` simultaneously, only one instance executes the DDL:

- `RunMigrateWithLock` → `pg_try_advisory_lock(8837291, 20260710)`.
- If the lock isn't acquired, retry every 500ms until `MIGRATION_LOCK_TIMEOUT_SECONDS` (default 120s).
- Drivers such as MySQL **skip** the advisory lock (single-instance migration by convention).

See `MIGRATION_LOCK_DESIGN.md` for details.

Startup: when `MIGRATION_RUN_ON_STARTUP=true` (default), migration runs with the lock held in `main.go`.

## Worker Leases (Per-Task Leader)

**Not** a global leader election; instead, **at most one holder per task**:

| Mechanism | Description |
| --- | --- |
| `locked_by` + `locked_until` | Atomic claim after BRPOP |
| Heartbeat renewal | Refresh `locked_until` at TTL/3 |
| Reaper | Reclaims expired leases, emits `lease_expired` event |
| `lock_version` | Optimistic concurrency control |

Applies to: collect, image, order_sync, customer_message_sync, product_publish, inventory_sync.

The same task is never executed by two workers simultaneously; different tasks run in parallel without lock contention.

## Worker Instance Registration (`worker.Registry`)

When `WORKER_HEARTBEAT_ENABLED=true`:

- Each instance writes to `worker_instances` (type, hostname, metadata).
- `last_heartbeat_at` is updated periodically; `MarkStaleWorkers` flags stale entries.
- `/health` aggregates `workers running` and queue depth.

When `WORKER_HEARTBEAT_ENABLED=false`, a `workerId` is still generated and used for task leases, but the instance table is not populated.

## In-Process Leader Mode (Lightweight)

**Task alert scanning**, `TASK_ALERT_SCAN_ENABLED`:

- Each process can register `worker.TypeTaskAlertScan`.
- Whether it actually runs is gated by the `taskcenter.enable_alert_scan_worker` setting.
- Scanning uses a context timeout, `TASK_ALERT_SCAN_LOCK_TTL_SECONDS` (default 120s), not a PG advisory lock; multiple instances may scan redundantly, but the alert-row `upsert` is idempotent.

Strict single-leader alert scanning could be implemented in the future by reusing the advisory lock or via Redis SETNX.

## Redis and Idempotency

- Only one instance's consumer receives a message via `BRPOP`; the DB claim prevents double execution after a crash.
- A second layer: `idempotency_records` and domain-level unique indexes.

## Deployment and Anti-Patterns

- Required: Redis queue, `WORKER_REAPER_ENABLED=true`, `MIGRATION_RUN_ON_STARTUP=true`.
- Do not disable the queue in favor of synchronous in-API processing under multi-instance deployment; do not disable the Reaper and then force-kill processes (tasks get stuck in `running`).
- MySQL has no advisory locks; multi-instance deployments must coordinate a single migration externally.

Related: `MIGRATION_LOCK_DESIGN.md`, `TASK_RELIABILITY_DESIGN.md`.
