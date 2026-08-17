---
name: code-quality
description: The single, complete master standard for TradeMind's project-wide code quality — automatic scope, lightweight checks, deep review, Baseline/Ratchet, and CI gating
---

# TradeMind Project-Wide Code Quality Master Standard

This Skill is the single, complete master standard for TradeMind code quality rules. Other entry points may only reference this file; they must not duplicate a second complete set of quality rules.

Related specialized standards continue to be maintained independently and are referenced by this standard:

- Project-wide testing: `.agents/skills/project-testing/SKILL.md`
- Admin UI: `.agents/skills/frontend-design/SKILL.md`
- Admin E2E: `.agents/skills/admin-e2e-testing/SKILL.md`
- Frontend unit tests: `.agents/skills/frontend-unit-testing/SKILL.md`
- Backend testing: `.agents/skills/backend-testing/SKILL.md`
- API contracts: `.agents/skills/api-contract-testing/SKILL.md`
- Modular architecture: `.agents/skills/modular-architecture/SKILL.md`

## 1. Automatic Scope

This standard automatically applies to any code addition, modification, refactor, or bug fix. The user does not need to explicitly say "check code quality," "use the code-quality Skill," "do a code review," "improve maintainability," or "check security."

Automatic scope includes Admin, Collector, the Go backend, shared packages, scripts, tests, migrations, API contracts, GitHub Actions, Docker and environment configuration, Redis, queues, workers, schedulers, and third-party platform adapters.

Pure documentation changes may skip the full code quality check, but must still verify links and paths, command accuracy, whether the docs conflict with other rules, whether they contain sensitive information, and whether they create a duplicate rule system.

## 2. Lightweight Checks and Deep Review

### 2.1 Lightweight Code Quality Check

Every code change automatically triggers a lightweight check that confirms at least:

- Whether the change scope is minimal, and whether there is unrelated formatting, meaningless import reordering, or line-ending changes.
- Whether naming is accurate, types are clear, and whether meaningless `any`, `@ts-ignore`, or overly broad `eslint-disable` were introduced.
- Whether errors are swallowed, whether a Promise may go unhandled, and whether null/exception states are fully handled.
- Whether there is duplicate logic, dead code, debug code, `.only`, unjustified `.skip`, or leftover `console.log`.
- Whether sensitive information is leaked, whether necessary tests are missing, and whether affected tests were run.
- Whether the API or business behavior was changed, and whether extra requests or a clear performance regression were introduced.

### 2.2 Deep Code Quality Review

Deep review is automatically triggered by: authentication and permissions; product inventory; publishing and listing; third-party platforms such as Douyin Shop; payments or funds; database transactions; Redis locks; queues and background tasks; concurrency; file uploads; auth tokens; new third-party adapters; new shared foundational components; new shared services; changes to the API envelope; changes to shared types; changes to migrations; changes spanning more than three business modules; large refactors; a single file whose responsibilities keep visibly growing; and fixes for data loss, duplicate submission, or duplicate task issues.

Deep review adds the following on top of the lightweight check: transaction boundaries, idempotency, retry boundaries, timeouts, race conditions, data consistency, permission bypass, log redaction, third-party failure degradation, rollback strategy, cache consistency, lock release, goroutine lifecycle, database query complexity, module dependency direction, and test completeness.

Triggering deep review does not authorize large-scale refactoring; unless the task itself requires it, only report the risks and make the minimal necessary changes.

## 3. Quality Issue Severity Levels

### Critical

Must block completion: leaked production credentials or private keys; tests that might connect to the production database; tests that might call real platform write endpoints; SQL injection; command injection; path traversal; permission bypass; data corruption; uncontrolled concurrent writes; deadlocks; sensitive data in logs; a clear data race; duplicate execution of a publish or inventory operation.

### High

Blocks by default: new TypeScript errors; new Go compile errors; new lint errors; unhandled Promise rejections; external requests without a timeout; retries without a bound; partial transaction commits; missing idempotency; exceptions completely swallowed; missing validation on a critical state transition; API contract drift; a new feature with no tests at all; a bug fix with no reasonable regression test and no explanation.

### Medium

Needs a fix or an explicit explanation: significant duplicated code; a function with too many responsibilities; unnecessary `any`; unstable `rowKey`; React Hook dependency issues; inefficient duplicate requests; N+1 query risk; poor readability; insufficient error context; tests overly coupled to internal implementation.

### Advisory

Can be addressed later: naming could be further improved; local abstraction opportunities; a historically oversized file; non-blocking performance optimizations; documentation improvement suggestions.

The final report must be organized by severity level — not every suggestion should be written up as a blocker.

## 4. Baseline/Ratchet Strategy

Admin currently has pre-existing TypeScript errors. Do not attempt to fix all historical errors at once, do not turn all historical errors into a hard gate for every PR, and do not let code-quality block all future development once it goes live.

Baseline/Ratchet requirements:

1. Run the real Admin typecheck: `pnpm quality:baseline:admin-ts`.
2. Collect historical errors, stripping unstable line and column numbers.
3. The baseline records only the file path, the TypeScript diagnostic code, a normalized error message, and the occurrence count of the same error.
4. The baseline must not record absolute local paths, user directories, timestamps, temp files, or a random order.
5. Later checks allow historical errors to decrease or disappear.
6. Later checks forbid new error signatures, an increase in the count of an existing error, errors in new files, or new errors introduced in modified files.
7. The baseline must not expand automatically; updating it requires explicitly running `pnpm quality:baseline:admin-ts -- --update` with a stated reason.
8. Regular CI must not auto-update the baseline.

Baseline file: `tests/quality/baselines/admin-typescript.json`.

## 5. General TypeScript Rules

- Do not add meaningless `any`; do not abuse `unknown as` or non-null assertions.
- Do not use `@ts-ignore` to mask a real error; when unavoidable, prefer `@ts-expect-error` with a stated reason.
- Do not use overly broad `eslint-disable`.
- Types should describe real domain meaning; public functions should have clear inputs and outputs.
- Do not treat an API response as trusted data directly — external data must be validated or safely normalized.
- Promises must handle rejection; async function errors must not be silently lost.
- Avoid implicit string-enum drift; prefer the project's existing shared types instead of duplicating the same DTO.
- Do not change production type semantics just for tests.

Check new or modified files for `any`, `@ts-ignore`, `@ts-expect-error`, `eslint-disable`, empty catches, unhandled Promises, unbounded retries, and unnecessary type coercion. Scripts are only responsible for surfacing candidates — the AI must review them in context.

## 6. React/Umi/Ant Design Rules

Admin changes must simultaneously account for: no side effects during render; correct Hook dependencies; avoiding duplicate requests from effects; preventing state updates after a component unmounts; preventing duplicate submission from rapid clicks; submit buttons must have loading/disabled states; `rowKey` must be stable and unique; do not use an array index that can change as the primary key for a list.

Do not rely on Ant Design's internal private classes for logic; do not use broad global Ant style overrides; do not use `!important` to mask structural problems; do not create nested Forms; a Form instance must be connected to a real Form; state cleanup on Modal/Drawer close must be explicit; URL state and component state must stay in sync; deep links and refresh must be restored correctly; readonly/disabled semantics must be accurate; error/loading/empty states must not be missing.

Prefer reusing `TmPageContainer`, `SectionCard`, `MetricCard`, `OperationToolbar`, `TmProTable`, `EmptyState`, `AppDrawer`, and `layoutTokens`. UI details are defined by `.agents/skills/frontend-design/SKILL.md`, and browser regression is defined by `.agents/skills/admin-e2e-testing/SKILL.md`.

## 7. Node.js/Collector Rules

Environment variables must be validated; falling back to a production address when configuration is missing is not allowed. Network requests must have a timeout; retries must have a bounded count and backoff; external responses must be validated; Promise rejections must be handled; files/streams/handles/connections must be released.

Transformation logic should stay as pure functions where possible, producing deterministic results for the same input; amounts and prices must not use unsafe floating-point logic; time and timezone behavior must be explicit; do not log full third-party responses, Authorization headers, cookies, or tokens; do not use unbounded queues or unbounded recursive retries; tests must not access real external sites or platforms.

For the Collector, focus checks on env parsing, price normalization, quality scoring, scrape failure handling, timeouts, rate limiting, retries, invalid data, duplicate products, memory usage under large batches, and missing third-party fields.

## 8. Go Backend Rules

All errors must be handled; error wrapping must preserve context — do not overwrite the original error with a context-free error string. `context.Context` must be propagated downward; external HTTP requests must set a timeout; response bodies must be closed.

Goroutine lifecycles must be controllable — do not start background goroutines that cannot be stopped; channel-close responsibility must be explicit; shared state must be synchronized; avoid data races.

Transactions must explicitly commit/rollback, with rollback errors handled sensibly; avoid N+1 queries; batch operations should account for pagination and upper bounds; Redis locks must have an expiry and verify ownership on release; queue jobs must be idempotent; retries must have a maximum count; third-party platform failures must not be mismarked as success.

DTO validation belongs at the correct boundary; handlers should not carry large amounts of domain logic; repositories should not decide business state machines; services should not depend on HTTP details; logs must not leak credentials; do not use panic to handle recoverable business errors; do not ignore JSON encode/decode errors; time handling should follow the project's standard consistently.

Automated quality checks should preferentially reuse `gofmt`, `go test`, and `go vet`. Reuse staticcheck or golangci-lint only if the repo already uses them; this project does not currently add large new Go lint tooling.

## 9. HTTP/API Rules

API changes must check method, URL, path params, query, payload, response envelope, error envelope, enums, nullability, pagination, permissions, readonly state, idempotency, and compatibility.

Give particular protection to multi-platform drafts, Douyin Shop platform drafts, the traditional `publishProduct`, readiness blocking, publication refresh, SKU binding, and inventory sync. Do not change API methods, payloads, routes, permissions, readonly state, state machines, or reload semantics under the guise of a refactor.

Third-party platform adapters must have a timeout, a bounded retry, must distinguish HTTP errors from business errors, must validate response structure, must correctly handle token expiry and rate limits, must not mistake failure for success, and must not log the full token. Tests use a fake adapter or mock server — never access a real platform. API changes automatically pull in `.agents/skills/api-contract-testing/SKILL.md`.

## 10. Database and Transaction Rules

Check whether migrations are repeatable, whether they depend on manual data, whether transaction boundaries are complete, whether partial failures roll back, whether unique/foreign-key constraints match business rules, whether nullability is accurate, whether state fields express real state, whether queries are N+1, whether bulk updates have limits, whether pagination and ordering are stable, whether optimistic locking or concurrency control is needed, whether duplicate requests could cause duplicate writes, whether JSON/JSONB fields lack structural validation, whether delete behavior produces orphaned data, and whether repositories leak ORM details into the domain layer.

Any migration change automatically triggers migration tests, database integration tests, repository tests, API contract checks, and deep quality review. Tests must not fall back to a dev or production database.

## 11. Redis/Lock/Queue Rules

Check Redis key namespacing, TTLs, cache invalidation, cache stampede risk, distributed lock expiry, lock release token/owner verification, task idempotency, safety of duplicate messages, retry ceilings, dead-letter observability, that failed tasks are not marked successful, whether a "running" state can get permanently stuck, whether stopping a worker releases resources, whether the scheduler double-registers, whether time-based logic is testable, and whether logs include taskId/traceId without logging sensitive payloads.

Redis/queue changes automatically trigger backend tests, Redis integration tests, queue tests, idempotency tests, retry tests, and deep quality review.

## 12. Concurrency and Async Rules

Async entry points must have a defined lifecycle, cancellation, timeout, and error-propagation strategy. Concurrent writes must have synchronization or an idempotency boundary. Background tasks must be stoppable, and repeated starts must be safe. Frontend async requests must avoid a stale response overwriting newer state; backend goroutines must be able to exit along with the context or process lifecycle.

## 13. Error Handling Rules

Prohibited: empty catches; catching and completely ignoring an error; only doing `console.log(error)`; returning just "failed" while losing context; logging the same error repeatedly at multiple layers; conflating user-facing errors with internal errors; exposing a stack trace to the frontend.

Errors must include appropriate context at minimum: operation, entity ID, task ID, trace ID, provider/platform, retry count. They must not include passwords, tokens, cookies, private keys, sensitive payment data, or unnecessary personal data.

## 14. Logging and Observability Rules

Log levels must be reasonable: debug for diagnostic detail, info for normal business state, warn for recoverable anomalies, error for failures that need handling. Do not log every normal business validation failure as an error. Do not log full third-party responses, Authorization headers, cookies, tokens, or sensitive user data.

## 15. Security and Sensitive Information Rules

Prefer reusing GitHub secret scanning and existing platform capabilities. This project also runs a lightweight, high-confidence local changed-diff scan for: private key headers, GitHub classic tokens, AWS access keys, OpenAI-style high-confidence secrets, obvious JWTs, database URLs with embedded username/password, hardcoded Authorization Bearer tokens, hardcoded cookies, and assignments to production password variables.

The scan only checks new or modified content, outputs file and line number, must redact secrets (never output the full secret), and must not upload scan result artifacts. Example values must be obviously fake; a plain UUID must not be treated as a secret. Finding high-confidence sensitive information must exit non-zero.

## 16. Performance Rules

Do not introduce obvious N+1 queries, unbounded pagination, unbounded queues, unlimited retries, request storms, or logging of large objects in full. Batch processing must have an upper bound and pagination; external requests must have a timeout; caches must have a consistency and invalidation strategy. Classify performance suggestions by severity — do not write non-blocking optimizations as hard blockers.

## 17. Test Code Quality Rules

code-quality applies equally to test code. Test names must describe real behavior; tests must not depend on execution order; must not share mutable global data; must not use long sleeps; must not use random, non-deterministic data; must not swallow test exceptions; must not use `.only`; must not use `.skip` casually; must not weaken assertions; must not use a broad console allowlist; must not connect to real services; fixtures should be minimal and valid; mocks must match the real contract; failure messages must be readable; helpers should have a single responsibility; do not create a giant catch-all test file.

## 18. Diff Hygiene

Before completing a task, check whether unrelated files were modified, whether a file was fully reformatted, whether line endings changed, whether meaningless import reordering was introduced, whether generated files/logs/test artifacts were introduced, whether the lockfile was accidentally modified, whether existing user changes were deleted, and whether debug code, unexplained TODOs, `.only`, `.skip`, or broad allowlists were left behind.

Prohibited: `git add .`; repo-wide unrelated formatting; rewriting unrelated modules just to pass quality checks; expanding scope under the guise of "opportunistic cleanup."

## 19. Dependency and Configuration Quality

New dependencies must include a justification for necessity; do not blindly install a large platform or a second lint/typecheck tool just to satisfy a quality gate. Configuration changes must avoid duplicate CI runs and conflicting rules, and must not introduce real credentials, absolute paths from a developer machine, or platform-private caches.

## 20. Quality Requirements for New Features

New features must have clear boundaries, types, error handling, tests, and affected quality checks. High-risk new features must undergo review of idempotency, permissions, transactions, retries, timeouts, log redaction, and rollback/failure semantics.

## 21. Quality Requirements for Bug Fixes

Bug fixes should prioritize adding a regression test; the fix must be minimal and target the actual root cause. Do not opportunistically refactor unrelated modules, and do not mask failures via skip, ignore, a broad allowlist, or weakened assertions.

## 22. Quality Requirements for High-Risk Modules

Authentication and permissions, publishing, inventory, third-party platforms, the database, Redis, queues, workers, schedulers, file uploads, tokens, the API envelope, shared types, migrations, and cross-module refactors must undergo deep review, with findings reported as Critical/High/Medium/Advisory.

## 23. Coordination with project-testing

Any code change must select affected tests per `.agents/skills/project-testing/SKILL.md`. `quality:affected` handles quality-check selection, `test:affected` handles test selection; they complement each other and are not interchangeable.

## 24. Coordination with frontend-design

Admin UI changes must simultaneously follow `.agents/skills/frontend-design/SKILL.md`. code-quality covers types, errors, async, security, diff hygiene, and test quality; UI visuals, layout, responsiveness, and shared component details are defined by frontend-design.

## 25. Coordination with admin-e2e-testing

Any change involving Admin pages, interactions, write requests, responsiveness, routing, state, or E2E files must follow `.agents/skills/admin-e2e-testing/SKILL.md`. code-quality does not duplicate the full E2E standard — it only requires that necessary browser regression and write-request safety are not skipped.

## 26. Modular Boundaries

code-quality is responsible for surfacing modularity risks: a file with too many responsibilities; pages/API/state machines mixed into a single file; a handler carrying domain logic; a repository deciding business rules; UI depending directly on low-level API implementation; cross-layer circular dependencies; duplicated DTOs/platform checks/state mapping in multiple places; a shared module depending back on a business module.

When a change involves a new module, a cross-module dependency, shared/common code, an adapter, a worker, a repository, a migration, a public API/type, or a large refactor, read `.agents/skills/modular-architecture/SKILL.md`. The complete rules for module boundaries, circular dependencies, cross-app dependencies, and the Architecture Baseline/Ratchet are defined there.

Do not force a file split based on line count alone. Only recommend triggering a dedicated modularity review when a new business domain is added, a new platform adapter is added, more than three modules are touched, a large component keeps growing, there is a circular dependency, a shared layer's responsibilities are unclear, or a new worker/queue/scheduler system is added. Small fixes must not be forced into large-scale modularization.

## 27. Prohibited

Prohibited: real credentials; production DB/Redis; real platform write endpoints; SQL/command injection; path traversal; permission bypass; sensitive data in logs; unbounded retries/queues/concurrency; empty catches; broad ignore/skip/allowlist; unrelated refactors; duplicate CI; auto-expanding the baseline; committing/pushing without the user's request.

## 28. Handling Test or Check Failures

On failure, locate the actual root cause first, distinguishing between a production defect, a test defect, an environment error, a historical baseline entry, a tool misconfiguration, and flakiness. Do not bypass directly, and do not auto-expand the baseline. Checks that cannot be run must state the command, the blocking reason, and the risk — never claim a skipped check passed.

## 29. Completion Report Format

The final report must list: current branch; starting workspace state; audit conclusion; count of Admin's historical TypeScript errors; Collector/Go status; new/modified files; automatic trigger scope; lightweight/deep conditions; Critical/High rules; baseline path and normalization method; baseline update command; how new errors are blocked; quality scripts; CI triggers; whether new dependencies were added; whether production code was modified; actual results of commands run; reasons for any skips; Critical/High/Medium/Advisory findings; diff stat; currently uncommitted files; whether sensitive files/test artifacts exist; and whether the work is ready for sign-off/commit/push.
