# Changelog

All notable changes to TradeMind are documented here.

## Unreleased

### Database schema naming (2026-08-10)

- Replaced phase-numbered `p9_*` and `p10_*` table names with stable inventory, SKU binding, platform credential, OAuth, and production-control domain names.
- Added a transactional, fail-closed PostgreSQL upgrade that renames existing tables and their indexes, constraints, triggers, and immutable-record function without copying or deleting data.
- Kept API routes, permissions, state machines, and the fail-closed P10 `L0` runtime boundary unchanged.

### Admin theme (2026-08-09)

- Added an icon-only, tooltip-labelled top-navigation light/dark theme switch with light mode as the default and local preference persistence.
- Applied Ant Design theme tokens across shared Admin chrome, login, dashboard, status surfaces, and responsive regression coverage.
- Moved the complete desktop brand and its sider toggle into the fixed top header, added permission-aware navigation search with a compact mobile entry, kept a compact mobile brand beside the menu trigger, removed the duplicate sidebar brand, and made the navigation drawer opaque above scrolled content.
- Made theme switching atomic and fully reversible for header, elevated, and portal surfaces, and safely centered mobile login and registration layouts.

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
