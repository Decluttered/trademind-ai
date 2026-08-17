---
name: frontend-unit-testing
description: TradeMind Admin TypeScript unit testing, React component testing, hooks, request transformation, and shared UI testing standard
---

# TradeMind Frontend Unit and Component Testing Standard

## Automatic Scope

Automatically applies to work involving TypeScript, React components, hooks, services, constants, URL state, request payloads, state mapping, shared UI, or test configuration within `admin/src/**`.

## Tech Stack

- Runner: Vitest.
- Environment: `jsdom` for React components; the same config can be reused with a Node environment for pure utilities.
- Component testing: React Testing Library, `@testing-library/jest-dom`, `@testing-library/user-event`.
- Do not introduce Jest, do not introduce Cypress. MSW is used only when a genuine module-level network mock is required.

## Test Targets

Prioritize coverage of:

- API envelope unwrapping and business errors.
- URL tab/section/query parsing and write helpers.
- Readiness state, results, grouping, and severity mapping.
- Publish/draft/batch payload construction.
- rowKey fallback, status tags, long-text display.
- Write Guard or test-utility pure logic.
- Baseline behavior of shared UI such as `TmPageContainer`, `EmptyState`, `StatusTag`.

## Environment Setup

The Vitest setup must handle:

- The `@` alias pointing to `admin/src`.
- Minimal mocks for `@umijs/max`'s `request`, `history`, etc.
- CSS/LESS imports.
- `matchMedia`, `ResizeObserver`, `IntersectionObserver`, localStorage, history, URLSearchParams.
- Async behavior caused by Ant Design Portals/animations.

Do not mask real errors with overly broad global mocks; only mock the Umi runtime, missing browser APIs, and external network boundaries.

## Assertion Principles

- Test user-visible text, roles, class/structure boundaries, and real business output.
- Do not test Ant Design's internal DOM.
- Do not write meaningless snapshots.
- Failure messages should be able to pinpoint the business behavior.

## Test Selection for Changes

- Pure TS utilities: related unit tests + type/build checks.
- Components/pages: related unit/component tests + Admin E2E smoke or affected specs.
- Service/request/payload: service unit tests + API contract checks + affected E2E.
- Shared UI/layout: component tests + Admin smoke + related responsive/overflow E2E.

## Bug Regression

For frontend bug fixes, prioritize adding a test that reproduces the user-visible issue first, then make the minimal fix. If it cannot be automated, report the reason.
