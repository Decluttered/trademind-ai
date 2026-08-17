---
name: backend-testing
description: Testing standard for TradeMind's Go/Gin/GORM backend — unit, HTTP, PostgreSQL, Redis, queues, state machines, and third-party adapters
---

# TradeMind Backend Testing Standard

## Automatic Scope

Automatically applies to work involving `backend/**`, Go handlers/services/models/repositories/providers, the database, Redis, queues, workers, schedulers, auth, permissions, state machines, or backend CI. Backend production and test code quality is governed by `.agents/skills/code-quality/SKILL.md`.

Tests for handlers/services/repositories/adapters/workers should verify module boundaries, transaction ownership, idempotency, and cross-layer call risks; detailed boundaries, circular dependencies, and the Architecture Baseline/Ratchet are defined in `.agents/skills/modular-architecture/SKILL.md`.

## Tech Stack

- Language: Go 1.25, module `github.com/trademind-ai/trademind/backend`.
- HTTP: Gin.
- ORM: GORM; PostgreSQL by default, with a MySQL driver also present.
- Redis: go-redis v9, LIST-based queues + workers.
- Testing: standard `go test`, `httptest`, `testify`. No second, conflicting framework.

## Unit Tests

Cover pure domain/service/provider logic: normal cases, invalid arguments, resource not found, insufficient permissions, disallowed state, external dependency failure, transaction rollback, idempotency, duplicate requests, error envelopes, and boundary values.

Key modules: auth, adminperm, product, SKU, pricing, inventory, productcheck readiness, productpublish, Douyin draft/SKU binding, image/files, taskcenter, idempotency, queue/worker, webhook, observability.

## HTTP Integration Tests

Use Gin + `httptest`. Test real routes, middleware, auth, DTOs, handlers, services, error handlers, and the response envelope. Third-party platforms must be faked/stubbed — never access real platforms.

Key endpoints: auth/profile, image/providers, product detail, readiness, inventory, publications, publication SKUs, publish targets, Douyin create-draft, traditional publish.

## PostgreSQL Integration Tests

Must pass the safety guard: the database name or URL contains `test`, `_test`, or `e2e`, the environment is `test`, and production domains/database names/default dev databases are forbidden. If no safe test database is available, the test should explicitly skip or fail with a clear reason — never fall back to the dev database.

Cover AutoMigrate running against an empty database, key tables/indexes/constraints, repository CRUD, unique/foreign-key constraints, transaction rollback, pagination, ordering, concurrency, idempotency, and JSON/enum/state fields.

## Redis / Queue Tests

Must pass the safety guard: `TEST_REDIS_URL`, a test DB number, or a test key prefix — never connect to production/dev business Redis. Cover cache set/get/expire, miss, invalidation, lock/idempotency, enqueue, consume, retry, failed tasks, duplicate tasks, and state transitions.

## Background Tasks and State Machines

Queue/worker/scheduler/cron/background tasks need coverage for created/running/success/failure/retry/cancel/timeout/duplicate/idempotency/illegal transitions/external platform failure/transaction failure. Use a fake clock or a short context for time-based logic — no long sleeps.

## Third-Party Adapters

Douyin, TikTok, Shopee, Lazada, Amazon, AI, image, storage, OCR, email, and collector integrations all use a fake server/stub. Cover 4xx/5xx/timeout/invalid JSON/missing fields/rate limiting/token expiry/signature failure/retry boundaries.

## Commands

- `pnpm test:backend`: Go unit tests.
- `pnpm test:backend:integration`: PostgreSQL/HTTP integration tests (requires a safe test database).
- `pnpm test:db`: database migration/constraint tests.
- `pnpm test:redis`: Redis/queue integration tests (requires a safe test Redis).

## Prohibited

Do not modify business logic just to make tests easier. Do not connect to real services. Do not use skip/only to mask failures. Do not use real credentials or real stores.
