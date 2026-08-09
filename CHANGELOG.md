# Changelog

All notable changes to TradeMind are documented here.

## Unreleased

### Production maintenance cleanup (2026-08-09)

- Removed historical phase gates, load-test harnesses, generated evidence, one-off acceptance scripts, and local Playwright/test outputs from the working tree.
- Removed residual P6/P7 verification commands, unreferenced backend placeholders, a one-off Admin codemod, and unused Admin/Collector symbols.
- Deduplicated the Admin brand image, aligned Collector Playwright dependencies, and added tracked documentation-path checks to CI.
- Kept GitHub Actions and their frontend, collector, backend, contract, architecture, PostgreSQL, Redis, and Admin E2E regression dependencies.
- Replaced the historical PostgreSQL phase wrapper with direct, isolated CI inventory integration commands.
- Adopted GitHub Actions for automated regression and human sign-off for product acceptance.
- Documented that a local test database is optional and is not recreated automatically; CI provisions isolated service containers.

## v0.1.0

- Initial TradeMind monorepo foundation with Go backend, React Admin, Node collector, PostgreSQL, Redis, Docker Compose, Provider abstractions, and open-source governance.
