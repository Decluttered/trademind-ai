---
name: admin-e2e-testing
description: Admin Playwright CI regression, write-request safety, responsive coverage, and manual acceptance rules for TradeMind's production maintenance phase
---

# TradeMind Admin Automated Testing and E2E Testing Standard

This Skill is the single, complete source of truth for Admin automated testing, E2E testing, Playwright MCP dynamic acceptance, Playwright Test persistent regression, CI triggers, and the rules for adding new page tests. Other entry points only reference this file; they do not duplicate the full standard.

## Production Maintenance Strategy (highest priority)

- Playwright Test continues to serve as the persistent CI regression suite under `admin/e2e/**` and `.github/workflows/admin-e2e.yml`; these tests and their mocks/fixtures must be preserved.
- Final acceptance of functionality, visuals, copy, responsiveness, and business flows is done by humans; full E2E defaults to being handled by GitHub Actions.
- Playwright MCP is used only when the user explicitly requests local browser acceptance testing or the task genuinely requires interactive diagnosis; it does not produce long-retained evidence.
- Wherever this Skill states a test "must be run" further below, that means the corresponding case must be covered by CI unless local execution is explicitly requested. Do not claim a test passed without running it locally.
- Non-GET request interception, the five-tier viewport matrix, state coverage, root-node overflow checks, and real-platform isolation requirements continue to be strictly enforced.
- Do not create phase/batch gates or one-off acceptance wrappers; Playwright reports, screenshots, and temporary results are cleaned up after diagnosis and are not committed to Git.

## 1. Automatic Scope

This Skill automatically applies to any task involving the Admin app: new pages, modifications, UI fixes, components, page layout, responsiveness, tables, forms, Modals, Drawers, Popconfirms, Tabs, routing, URL params, deep links, loading/empty/error states, readonly/disabled states, write operations, request payloads, duplicate-submission prevention, console warnings, accessibility, shared UI components, `global.less`, `TmPageContainer`, or `layoutTokens`.

The user does not need to explicitly say "run E2E," "use Playwright," or "use the testing Skill." The AI must automatically recognize and execute the relevant tests. Pure service, type, or utility function changes may skip browser testing only after confirming that DOM, className, page state, user interaction, routing, request trigger timing, loading/error/empty states, responsiveness, and write-request payloads are all unaffected.

## 2. Test Layers

- Exploratory acceptance: Playwright MCP, used during development to dynamically confirm real UI, console, network, and interaction behavior.
- Persistent regression: Playwright Test, for repeatable runs within the repo, CI, and scheduled regressions.
- Static checks: `pnpm check:dev`, `pnpm check:ui-copy --strict`, `pnpm build:admin`, and any necessary type/build checks.

## 3. Playwright MCP Mode

During development, Playwright MCP can access `http://localhost:8001`. If the user has already started the Admin dev server, reuse the existing service — do not start, stop, or kill processes. All non-GET APIs must be intercepted with `browser_route` first; real platform write operations must never be executed.

## 4. Playwright Test Mode

Persistent E2E uses Playwright Test, with Chromium as the first-phase browser. The default `baseURL` is `http://127.0.0.1:8001`; locally it reuses an existing server, while in CI the `webServer` config starts `pnpm dev:admin` on port 8001. Tests must not depend on the backend service — all `/api/v1/**` business endpoints are provided via mocks.

## 5. Test Directory Layout

Admin E2E lives under `admin/e2e/`:

- `fixtures/`: test fixtures.
- `mocks/`: API envelope and business mocks.
- `pages/`: stable Page Objects.
- `specs/`: tests split by feature.
- `utils/`: network guard, console guard, assertion, and routing utilities.

Do not cram all tests into a single file, do not put business mocks in the Playwright config, and do not create a giant catch-all helper.

## 6. Test Data Rules

Test data must use an explicit `e2e` or `mock` prefix, e.g. `e2e-user`, `e2e-product-draft`, `e2e-shop-douyin`, `e2e-publication-old`, `e2e-publication-new`. Do not reference production IDs, real stores, real users, or real tokens. Each test sets up its own mocks independently and must not depend on execution order.

## 7. API Mock Rules

Mocks must be based on the real frontend request helper and service types. The unified envelope is:

```ts
{ code: number; message: string; data: T; traceId?: string }
```

`code !== 0` indicates a business error. The `data` field for `GET /api/v1/image/providers` must be `ImageProviderCapability[]`, not `{ data: { list: [] } }`.

## 8. Write-Request Safety Boundary

All non-GET API requests are blocked by default unless explicitly allowed by the current test. At minimum, recognize `POST`, `PUT`, `PATCH`, and `DELETE`. The Write Guard must capture method, URL, path, query, payload, count, and order; any undeclared write request must be blocked and must fail the test.

Allowed write endpoints must be declared per test and must return mocked success/failure responses. The guard must support asserting zero requests on cancel, exactly one request on confirm, exactly one request even after rapid repeated clicks, and no extra write requests.

## 9. Selector Rules

Prefer `getByRole`, `getByLabel`, `getByPlaceholder`, explicit `getByText`, and stable business identifiers. Do not primarily rely on Ant Design's internal classes, deep CSS selectors, `nth-child`, coordinate-based clicks, random IDs, or incidental DOM hierarchy. Add `data-testid` minimally, and only when no reliable semantic selector exists; the final report must list any production code changes made for this purpose.

## 10. Console and Runtime Error Rules

The Console Guard must capture `pageerror`, `console.error`, unhandled rejections, React fatal errors, Ant Design fatal warnings, and the HMR overlay. By default, `pageerror`, `console.error`, new React warnings, and new AntD warnings all fail the test. Any allowlist must be precise and justified — do not use `/warning/`, and do not blanket-ignore all React warnings or all AntD warnings.

Current candidate warnings (only allowlist precisely if they remain consistently present): `useForm is not connected to any Form element`, `Each child in a list should have a unique "key" prop`.

## 11. Responsive Rules

Required viewports: 1440x900, 1280x800, 1024x768, 768x900, 375x812. The page root must have no horizontal overflow, the left/right edge deviation between Header and Content must not exceed 4px, Tabs must remain usable, the main action area must not overflow the viewport, and tables must scroll only within their own container.

## 12. Page Root Overflow

Standard assertion:

```js
document.documentElement.scrollWidth <= document.documentElement.clientWidth;
document.body.scrollWidth <= document.body.clientWidth;
```

On failure, the assertion must output both the actual and expected values.

## 13. Routing, Deep Links, Refresh, and History

Routing, URL state, tabs, sections, deep links, and refresh recovery must be tested. The normal product-detail tabs currently use `replaceState`; tests must not incorrectly require `pushState`. Invalid tabs must fall back safely.

## 14. Modal / Drawer / Popconfirm

Must verify title, context, default values, loading, confirmLoading, cancel, close, mobile width, and that a single confirm sends only one request. Cancel must not send a request; confirmation must be preserved for dangerous operations.

## 15. Loading / Empty / Error / Readonly

New or modified pages must cover normal, loading, empty, error, readonly, disabled, and submitting states. Errors must not be disguised as empty data, empty data must not be disguised as an error, and readonly must not expand or change the original business semantics.

## 16. Test Priority: P0 / P1 / P2

P0: run on every Admin PR. Covers Admin smoke tests, core routes, the seven product-detail tabs, a reduced responsive/overflow check, publish request safety, core API contracts, and console fatal errors.

P1: run before merge or daily. Covers the full five-tier responsive matrix, Basic save, AI tasks, image tasks, SKU editing, inventory adjustment, alert thresholds, inventory sync, readiness checks, Douyin Shop configuration, mapping, uploads, bind/unbind, readonly, and Modal/Drawer.

P2: run nightly or manually triggered as a full regression. Covers all Admin routes, long text, all states, history, permission combinations, console warning audits, baseline accessibility scans, and baseline performance checks.

## 17. Change Type and Test Selection

- Local page TSX/LESS changes: P0 smoke, the target page's spec, and related responsive checks.
- `TmPageContainer`, `layoutTokens`, `global.less`: P0 smoke, base routing across all pages, Header/Content alignment, and the five-tier overflow check.
- DraftDetail: product-draft, publish-safety, responsive; add contract tests if the envelope is affected.
- MultiPlatformPublishCenter: publish-safety, DraftDetail publish smoke, Console guard.
- Routing, tabs, sections, history: navigation, deep-link, refresh restore, history.
- Forms, Modals, Drawers: the corresponding interaction spec, zero requests on cancel, one request on confirm, 375px.
- Service or response envelope changes: contract tests and smoke tests for affected pages.
- Copy-only changes: `check:ui-copy --strict` and smoke tests for the target page.
- Backend-only changes with no Admin impact: full Admin E2E is not required, but run the corresponding backend tests.

## 18. Automatic Trigger Rules

The AI must not skip relevant tests because they are slow. Work must not be declared complete without running the necessary tests; if tests are blocked, the reason, the blocking command, and the first root cause must be stated.

## 19. CI Rules

PRs, dev pushes, workflow_dispatch, and scheduled regressions should trigger Admin E2E. CI uses real Node/pnpm versions and runs `pnpm install --frozen-lockfile`, `pnpm exec playwright install --with-deps chromium`, static checks, and P0 E2E. CI does not connect to a production database, real Redis, real platforms, real stores, or real APIs.

## 20. Requirements for New Pages

New Admin pages must simultaneously ship with routing smoke tests, auth mocks, normal/loading/empty/error/readonly coverage, desktop and 375px viewports, root-node overflow checks, console guard, mocks for all write requests, zero requests on cancel, single submission, key payload assertions, and URL state refresh recovery. Do not add a new page without adding its automated tests.

## 21. Requirements for Modified Pages

Modifying an existing page requires identifying the affected specs and prioritizing updates to existing tests rather than creating new duplicate files. Add regression scenarios that verify old behavior, new behavior, unchanged write requests, responsiveness, console state, and no extra write requests. For bug fixes, prioritize adding a stable regression test that fails before the fix and passes after.

## 22. Request Payload Contract Tests

Write requests must assert method, URL, path params, query, payload, request count, and order. Key scenarios include creating a multi-platform draft, creating a Douyin Shop product draft, the legacy `publishProduct`, inventory sync, SKU binding, and form save. Do not modify the API/payload to accommodate an incorrect test.

## 23. Handling Test Failures

On failure, first locate the actual root cause. Fix the mock if the mock structure is wrong; if it is a genuine production defect, record it, and only apply a minimal fix if it's an infrastructure blocker affecting test stability. Do not delete, skip, or weaken a failing test to mask a problem.

## 24. Prohibited

Prohibited: Cypress, Selenium, Puppeteer, or any second browser testing framework; production accounts; real stores; real API tokens; real platform write endpoints; CI connecting to the production backend; allowing undeclared write requests through; ignoring all `console.error`; ignoring all React/AntD warnings; broad `waitForTimeout` usage; coordinate-based clicks; `nth-child`; random test data; order-dependent tests; large-scale production code changes made just to satisfy tests; bulk-adding `data-testid`; auto-updating snapshots to mask regressions; skipping failing tests outright; declaring completion without running tests; committing/pushing without the user's request.

## 25. Completion Report Format

The final report must list at least: current branch, starting workspace state, audit results, whether Playwright already existed, dependencies, config path, E2E directory, Skill path, entry-point updates, Cursor rule, frontend-design references, CLAUDE.md status, package scripts, Network Write Guard, Console Guard, API envelope helper, Page Objects, assertions, P0 tests, pages/tabs/viewports/publish/API-contract coverage, GitHub Actions, trigger conditions, whether the real backend was accessed, whether real write operations were executed, run commands, pass/fail counts, warning allowlist, whether production code was modified and why, modified files, diff check, check:dev, check:ui-copy, build:admin, E2E smoke/contracts, diff stat, currently uncommitted files, remaining risks, whether future UI additions/changes will auto-trigger, and whether the work is ready for sign-off.
