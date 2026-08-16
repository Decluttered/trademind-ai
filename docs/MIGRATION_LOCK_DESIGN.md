# Database Migration Lock Design (P2)

> When multiple API instances start up in parallel, a **PostgreSQL Advisory Lock** guarantees `AutoMigrate` runs serially, avoiding DDL races.

## Implementation Location

- `backend/internal/database/migration_lock.go`
- Startup entry point: `cmd/server/main.go`

```go
database.RunMigrateWithLock(ctx, db, timeout, database.AutoMigrate)
```

## Advisory Lock Keys

Uses **two int32 values** as the `pg_advisory_lock` key (session-level lock):

| Key | Value | Description |
| --- | --- | --- |
| `migrationLockKey1` | `8837291` | Fixed project salt value |
| `migrationLockKey2` | `20260710` | P2 reliability phase identifier (2026-07-10) |

```sql
SELECT pg_try_advisory_lock(8837291, 20260710);  -- non-blocking attempt
-- after migration completes
SELECT pg_advisory_unlock(8837291, 20260710);
```

When `pg_try_advisory_lock` fails it does not block the connection; the application layer retries in a loop.

## Execution Flow

```text
Startup → MIGRATION_RUN_ON_STARTUP?
  ├─ true  → RunMigrateWithLock
  │           ├─ postgres: try lock → on success, AutoMigrate → unlock (defer)
  │           └─ on failure: retry every 500ms until timeout
  └─ false → AutoMigrate directly (no lock; suitable when migration is handled externally)
```

### Timeout

- `MIGRATION_LOCK_TIMEOUT_SECONDS` (default **120**).
- Context and lock wait share this timeout.
- Timeout error: `migration lock: timeout waiting for advisory lock`.

### Non-PostgreSQL

When `db.Dialector.Name() != "postgres"`, `run(db)` is called **directly**, without locking.

MySQL deployment convention: single-window migration or a CI-dedicated migration job.

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `MIGRATION_RUN_ON_STARTUP` | `true` | Run AutoMigrate at startup |
| `MIGRATION_LOCK_TIMEOUT_SECONDS` | `120` | Maximum time to wait for the lock |

Disabling startup migration (for large production changes):

```env
MIGRATION_RUN_ON_STARTUP=false
```

Migration must then be run **once** as part of the release process (the same advisory-lock script can still be used).

## P2 Migration Contents

`migrateReliabilitySchema` in the `AutoMigrate` chain:

- Tables: `idempotency_records`, `webhook_events`
- Indexes: customer-service `client_message_id`, webhook uniqueness, idempotency status/lease indexes

Independent of the Phase 10.2 order unique index and publish-batch index; all run sequentially within the AutoMigrate sequence.

## Operations

Rolling releases can start in parallel; a crashed holder's session ends and automatically releases the lock. Monitor `database_migrate_failed` and lock timeouts.
