---
name: modular-architecture
description: The single, complete master standard for TradeMind's modular architecture, module boundaries, circular dependencies, Baseline/Ratchet, affected-architecture checks, and CI gating
---

# TradeMind Modular Architecture Master Standard

This Skill is the single, complete modular architecture standard for the whole TradeMind project. Other Skills, AGENTS files, Cursor rules, CI, and docs may only reference this file — they must not duplicate a second complete set of architecture rules.

This standard follows the approach of "first describe the real architecture, then establish boundaries, then use a ratchet to prevent regression." This round does not require fixing all historical architecture debt at once, and architecture optimization must never be used as a pretext to change the API, payload, permissions, state machine, database semantics, or business behavior.

## 1. Automatic Scope

The following tasks must automatically read and follow this Skill; the user does not need to explicitly say "modularize," "design the architecture," or "check dependencies":

- Adding a new top-level business module, a complete page system, a domain model, a repository, or a database domain model.
- Adding a new third-party platform adapter, shared component system, shared service, or shared DTO/type/enum.
- Adding a new worker, queue consumer, scheduler/cron, or background task system.
- Modifying a migration, a cross-module API, shared/common/package code, or three or more business modules.
- Moving a large number of files, splitting a large page or large service, modifying a module's public entry point.
- The appearance of a circular dependency, a cross-layer call, or shared code depending back on a feature/page.
- A handler carrying a large amount of domain logic, a repository deciding business state, or a worker duplicating service logic.
- Scattered platform conditional checks, DTOs/state/enums duplicated in multiple places, a single file's responsibilities continuing to grow, or a large refactor.

## 2. Non-Automatic Scope

The following small changes generally do not trigger a full deep architecture review — only the code-quality lightweight boundary check applies:

- Button spacing, copy, local styling, a single null-value fix, a single type error, one table column.
- A single field mapping in an interface, a small internal refactor within one file, a test fix that doesn't change the public API, README doc edits.

However, as soon as a small change introduces a new cross-module import, a new shared dependency, a new public type, a new platform check, a new repository call, new worker logic, or a new circular dependency, the architecture check must still be triggered.

## 3. Lightweight Boundary Check

Every code change must at least confirm:

- Whether a new cross-app source dependency was added, or production code now depends on tests/e2e/mock/fixture.
- Whether shared/common code now has a new reverse dependency on pages, feature-internal implementation, handlers, repositories, or workers.
- Whether a new deep import into internal implementation was added, bypassing an existing public entry point.
- Whether an obvious new cross-layer call, runtime circular dependency, or duplicated public DTO/enum definition was introduced.
- Whether historical violations outside `tests/architecture/baselines/module-boundaries.json` were expanded.

## 4. Deep Architecture Review

When deep review is triggered, must check: module responsibilities, public API, dependency direction, circular dependencies, single source of truth for DTOs/types/enums, adapter/worker/repository boundaries, cross-app coupling, baseline/ratchet changes, and coordination with testing and code quality.

Triggering deep review does not authorize large-scale refactoring. Unless the task explicitly requires it and it has been confirmed, do not move large numbers of production files, split DraftDetail, rebuild the Go layering, or change business semantics.

## 5. Architecture Issue Severity Levels

### Critical

Must block completion: production code depending on test mocks; a frontend bundle pulling in backend or script private implementation; cross-app dependencies causing coupled production releases; a circular dependency causing initialization errors; shared code depending back on features forming a core cycle; a worker bypassing the business layer to perform a dangerous write; a handler bypassing permissions or the state machine.

### High

Blocks by default for new issues: a new runtime circular dependency; a new illegal cross-layer dependency; a new cross-app source dependency; new production code importing test/e2e/mock; a new handler -> repository layer violation; a new shared -> feature/page reverse dependency; new third-party structures leaking into the domain core; a new worker duplicating a business state machine; a new module with no clear ownership or public boundary.

### Medium

Needs a fix or an explanation: deep imports into internal implementation; duplicated DTOs; scattered platform conditional checks; a page with clearly too many responsibilities; service coupled to the transport layer; a repository returning a page-display structure; duplicated state mapping in multiple places; a shared module with unclear responsibilities.

### Advisory

Can be addressed later: historically large files; modules that could be split but aren't currently blocking; naming and directory structure that could be improved; a public entry point that could be made clearer; suggestions for future modularization.

The historical size of the DraftDetail file can only be Advisory — it must not be directly listed as High in this round.

## 6. Baseline/Ratchet

The architecture baseline file is `tests/architecture/baselines/module-boundaries.json`. The baseline only records stable, deterministic, machine-recognizable historical violations — it does not record local absolute paths, line/column numbers, timestamps, random ordering, file-length suggestions, or subjective opinions.

Rules:

1. New violation signatures block.
2. An increase in the count of an existing violation blocks.
3. A decrease in violations passes, with a suggestion that the baseline could be tightened.
4. The baseline does not expand automatically; CI does not update the baseline.
5. Updates must be run explicitly: `pnpm architecture:baseline -- --update`.
6. The baseline only manages historical debt — it does not imply the architecture is correct.

## 7. Application-Level Boundaries

Current real applications:

- `admin/`: React + TypeScript + Ant Design Pro admin app.
- `collector/`: Node.js + TypeScript + Playwright collection service.
- `backend/`: Go + Gin + GORM backend.
- `tests/contracts/`: source of API contract tests.
- `scripts/`: local, CI, and quality-test orchestration scripts.

Hard rules: Admin runtime code must not import Collector source; Collector runtime code must not import Admin source; the Go backend must not depend on frontend source; production apps must not import e2e/test/mock/fixture; tests may import production code; contracts/shared code may be depended on by apps but must not depend back on a specific app; scripts must not be depended on by browser runtime code; CI helpers must not enter the production bundle.

## 8. Admin Frontend Boundaries

`admin/src/pages/**` is responsible for route-level orchestration, composing feature/shared UI/services/hooks. It should not be depended on by shared UI, services, utils, or types, and should not carry the entirety of API calls, state machines, form logic, and view logic.

`admin/src/components/ui/**`, `TmPageContainer`, `SectionCard`, `MetricCard`, `OperationToolbar`, `TmProTable`, `EmptyState`, `AppDrawer`, and `layoutTokens` should remain domain-agnostic or have low domain coupling — they must not import pages, must not call specific business APIs directly, and must not be bound to a specific platform state machine or page-internal state.

`admin/src/services/**` is responsible for API calls and the transport boundary — it must not import pages, must not import visual components, must not contain page layout logic, must not depend on React component instances, must not decide complex business state machines, and its request/response types must be clear.

Generic utils must not depend on pages, React component instances, or specific feature-internal implementation unless placed inside that feature; they should stay deterministic, testable, and must not carry hidden global state.

Where a feature/domain structure already exists, code within a domain may depend on shared code; domains should interact through stable public entry points rather than forming implicit bidirectional dependencies. This round does not mandate migrating every page to a feature-based directory structure.

## 9. DraftDetail Special Rules

The product draft detail page is a historically complex module. This round does not directly split DraftDetail, does not change the seven Tabs, does not modify the business state machine or API/payload, does not move large numbers of files, and does not refactor production code purely to improve an architecture score.

Principles for future splitting: the page entry point should only handle routing and top-level orchestration; the seven Tabs should be independent modules; shared publish state and actions should have clear ownership; Douyin platform logic should live in a platform module; inventory/readiness/publish state should avoid implicitly modifying each other; request-payload construction should be separated from the UI; platform adaptation logic must not continue to be scattered.

## 10. Collector Boundaries

Collector is organized around the existing `collector/src/providers/**`, `browser/**`, `normalizer/**`, `tasks/**`, `types/**`, `config/**`:

- Source adapters handle source integration and must not modify global core rules.
- fetch/client handles networking, parser handles field parsing, normalize handles standardization, quality score handles scoring, dedup/output/persistence handles output, orchestration handles coordination.
- Network requests and business transformation are kept separate; scoring should stay a pure function as much as possible.
- Do not reimplement the same price normalization across multiple source adapters.
- Raw third-party structures are confined to the adapter boundary; external data is normalized at that boundary.
- Collector must not depend on Admin page code; test fixtures must not enter the production runtime.

## 11. Go Backend Boundaries

The current backend is organized around `backend/internal/modules/**`, `backend/internal/providers/**`, `backend/internal/api/**`, `backend/internal/rdb/**`, `backend/internal/queue/**`.

Target direction: handlers/controllers handle HTTP input/output, parameter parsing, and boundary validation, calling into service/usecase — they must not write complex SQL directly, must not carry a full business state machine, and must not manipulate third-party SDK details.

service/usecase handles business rules, state transitions, and transaction orchestration; it depends on repositories and abstract adapters, must not depend on HTTP request/response types, and must not generate page-display structures.

repository handles persistence — it does not decide whether a business state is allowed, does not depend on HTTP handlers, and does not carry third-party platform logic.

model/domain expresses domain entities and values — it does not depend on the transport layer or a specific database connection, and domain types should avoid being fully conflated with HTTP DTOs.

adapter/client isolates third-party capabilities such as Douyin Shop, TikTok, Shopee, Lazada, Amazon, AI providers, image providers, storage providers, OCR, email, and the collector; third-party response structures must not leak beyond the service layer, and adapters must not directly decide core business state.

worker/job calls service/usecase and does not reimplement business rules — it is responsible for task lifecycle and retry orchestration, and must not bypass the service layer to modify database state.

If the current package doesn't have a strict layered directory structure, this round does not mandate a large migration; new code should follow the target direction, and historical exceptions are managed via the baseline or as Advisory items.

## 12. API Contract Boundaries

This repo currently maintains the source of truth for the API via `tests/contracts/**`, the backend route/handler/service layer, and Admin service/mock contract tests. API contract changes must check method, URL, query, payload, response envelope, error envelope, enums, nullability, pagination, permissions, readonly state, and idempotency.

Contracts may be shared via OpenAPI, JSON Schema, contract fixtures, generated types, or stable enums, but do not duplicate a large amount of contract content without establishing a consistency test. The browser and Go sides are not required to directly share the same database structure type.

## 13. Shared/Common Boundaries

Shared code is a high-risk area. Any new shared/common addition must answer: Is it genuinely reused by multiple modules? Is it stable and generic? Does it carry business-domain concepts? Should it stay inside a feature instead? Does it create a reverse dependency? Does it expand the public API? Does it have tests? Does it cause cross-app coupling?

It is prohibited to put single-page-specific logic, single-platform-specific checks, single-endpoint-specific DTOs, temporary workarounds, a complex service called by only one module, a business state machine, or a database repository into shared code.

Shared code must not depend on pages, feature-internal files, HTTP handlers, repositories, workers, or E2E/test/mock code.

## 14. Third-Party Platform Adapters

Douyin Shop, TikTok, Shopee, Lazada, Amazon, AI providers, image providers, storage providers, OCR, email, etc. must be isolated through an explicit adapter/client.

Third-party SDKs and response structures are confined to the adapter layer; the service uses an internally normalized model; platform errors are converted to internal error types; platform conditional checks must not be scattered across pages, repositories, and workers; new platforms are added via a new adapter; do not keep growing a giant switch statement in a generic service; platform capability differences are expressed via a capability/config table; adapters do not directly decide user permissions or UI state; tests use a fake adapter.

Adding a new platform automatically triggers modular-architecture and code-quality deep review, plus backend-testing, api-contract-testing, and project-testing.

## 15. Worker/Queue/Scheduler

The Redis client wrapper lives in the infrastructure layer; business modules must not scatter raw Redis commands. Distributed lock wrapping is unified. Queue producers do not decide consumer business details. Consumers call service/usecase. Schedulers only trigger — they do not duplicate business logic. Job payloads use an explicit version or a stable structure. Jobs do not depend on HTTP handlers. Retry and idempotency policy is defined explicitly by the task module. A shared queue module must not depend back on a specific page or handler.

Adding a new worker/queue/scheduler automatically triggers a full architecture review.

## 16. Database and Repository

Migrations are only responsible for schema/data migration and must not depend on application runtime services. Repositories handle database access; handlers must not manipulate the database directly; workers must not bypass the service layer to perform business writes; database models must not directly become every API response; state-machine validation must not rely solely on a database enum; transaction boundaries are explicitly controlled by service/usecase; multi-repository operations must have clear transaction ownership. Migration changes must trigger deep architecture review.

## 17. Dependency Direction

Default direction: application entry/routing -> handler/page orchestration -> service/usecase -> repository/adapter/infrastructure -> model/domain/shared primitives. Shared/common/contract code can only be depended on by upper layers — it must not depend back on a specific app's implementation. Tests depend on production code; production code must not depend on tests.

## 18. Circular Dependencies

TypeScript/JavaScript circular dependency checks cover reusable modules within `admin/src`, `collector/src`, `scripts`, and other real TS workspaces, ignoring node_modules, dist, build, coverage, test-results, playwright-report, generated files, and test fixture graphs.

Runtime dependency cycles must be distinguished from type-only dependency cycles: runtime cycles block by default; purely type-only cycles are marked High or Medium based on risk and are not treated the same as runtime cycles.

Go import cycles are already blocked by the compiler, but the layered dependency direction still needs to be checked.

## 19. Cross-Layer Calls

Prohibited new additions: handler -> repository layer violations, worker -> handler, repository -> handler/service, shared -> pages/feature/internal, service -> private HTTP request/response types, adapter -> UI state, and similar directions. Historical exceptions that are machine-recognizable and stable may enter the baseline; otherwise they are listed as Advisory and narrowed down in future refactors.

## 20. Module Public API

Modules should preferentially access each other through stable entry points, e.g. `index.ts`, an explicit exported interface, a Go package's exported public API, or a contract schema. Do not deep-import another module's internal implementation, private helpers, page-internal components, test fixtures, or an adapter's internal response type directly from its internal directories.

Do not create a meaningless `index.ts` for every directory just for form's sake; public entry points should only be established where a module boundary genuinely needs one.

## 21. DTOs, Types, and Enums

Check whether API DTOs, platform enums, publish/readiness/inventory states, or string constant mappings are duplicated; whether a transport DTO is directly serving as the domain model; whether third-party DTOs are leaking outward; whether frontend display state is incorrectly reaching back to affect backend domain state.

Prefer establishing a single source of truth, but do not force the browser and Go sides to directly share source files that are fundamentally incompatible.

## 22. New Module Creation Standard

Before adding a new module, state its responsibility, owning app, boundaries, public entry point, dependency direction, DTO/enum source, test strategy, and affected checks. A new module must be expressible within the roots and rules of `tests/architecture/module-boundaries.json` — it cannot rely on a temporary, broad exemption.

## 23. Modifying Existing Modules Standard

Small changes should keep a minimal diff — a directory rebuild is not required. Cross-module changes must state the call direction and public entry point. When modifying a public API/type/enum, update the API contract, tests, and affected architecture checks together.

## 24. Large Files and Responsibility Growth

A large file is not automatically an error. It is only escalated to Medium/High when the file keeps taking on new business domains, new platform branches, new state machines, new repository calls, or a new public API. Historically large files are Advisory items, to be split gradually based on business risk.

## 25. Module Split Decision Criteria

A split is only recommended when most of the following hold: responsibilities can be clearly named; the call direction is stable; the split reduces cross-layer knowledge; test boundaries become clearer; business semantics are unchanged; no large, unrelated migration is required; a genuine public entry point exists.

## 26. Guarding Against Over-Abstraction

Prohibited: introducing unnecessary interfaces, factories, abstraction layers, empty index files, or catch-all shared services in the name of "modularization." Three similar calls should generally stay direct and clear, unless there is already genuine, stable cross-module reuse.

## 27. Coordination with code-quality

Any change that triggers this Skill must also follow `.agents/skills/code-quality/SKILL.md`. code-quality handles types, security, error handling, diff hygiene, and deep quality risk; modular-architecture handles module responsibilities, dependency direction, circular dependencies, and the baseline ratchet.

## 28. Coordination with project-testing

Any change that triggers this Skill must also follow `.agents/skills/project-testing/SKILL.md`. Architecture boundary script tests run via `pnpm architecture:test`; the affected gate runs via `pnpm architecture:affected`.

## 29. Coordination with frontend-design

Large page decomposition, shared UI boundaries, cross-page module design, and DraftDetail evolution must also follow `.agents/skills/frontend-design/SKILL.md`. UI visuals, responsiveness, and shared component details are not duplicated in this Skill.

## 30. Coordination with backend-testing

Changes to handler/service/repository/adapter/worker boundaries must be coordinated with `.agents/skills/backend-testing/SKILL.md`, covering the state machine, transactions, idempotency, Redis/queue, third-party fake adapters, and database integration.

## 31. Coordination with api-contract-testing

Changes to the API, DTOs, envelope, public types, contract fixtures, or Admin service/mocks must be coordinated with `.agents/skills/api-contract-testing/SKILL.md` to ensure frontend/backend contract consistency.

## 32. Prohibited

Prohibited: automatic commit/push; moving large numbers of production files without confirmation; changing business semantics to improve an architecture score; auto-expanding the baseline; using a broad allowlist to mask violations; introducing a large third-party architecture-scanning tool; production code depending on tests, mocks, e2e, or fixtures; the Go backend depending on frontend source; Admin/Collector runtime code importing each other.

## 33. Completion Report Format

The final report must list: current branch; starting workspace state; app and module audit; current structure of Admin/Collector/Go/Shared/Adapter/Worker; number of circular dependencies; number of cross-layer violations; number of cross-app violations; Skill path; trigger/non-trigger conditions; AGENTS/Cursor/other Skills updates; module-boundaries configuration; architecture docs; each boundary rule; TS cycle-detection implementation; Go dependency-check implementation; baseline path and count; baseline update command; how new violations are blocked; how violation decreases are handled; scripts and test files; test count; package scripts; quality/test affected coordination; CI changes; whether new dependencies were added; whether production code was modified; actual results of commands run; Critical/High/Medium/Advisory findings; list of modified files; diff stat; uncommitted files; sensitive-file/test-artifact check; whether the baseline was expanded; future trigger scenarios; and whether the work is ready for sign-off/commit/push.
