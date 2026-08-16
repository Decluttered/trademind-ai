# AI Coding Rules and Documentation Sync Requirements

This document constrains the basic engineering rules for developing TradeMind with an AI agent, Cursor, Copilot, or human collaboration. The core principle is: **code, configuration, documentation, examples, and CI must stay in sync**. For the more complete cross-tool execution flow, minimal context packages, token-saving practices, and the experience-accumulation mechanism, see [ai-workflow.md](ai-workflow.md).

## Basic Principles

- Understand existing module boundaries before modifying code.
- Prefer the repository's existing architecture and Provider abstractions; do not bypass the business layering.
- Never write API keys, tokens, secrets, cookies, or passwords into code, logs, examples, or screenshots.
- Do not default to introducing heavyweight architecture; in the MVP stage, prioritize being runnable, deployable, and maintainable.
- When modifying code, also consider tests, builds, deployment, documentation, and example configuration.
- For any cross-module change, check [module-map.md](module-map.md) first, then decide which files need to be kept in sync.

## Production Maintenance Testing Rules

- GitHub Actions is the sole continuously-executed entry point for automated regression; the core tests, fixtures, mocks, and configuration that workflows depend on must be kept.
- By default, run relevant static checks, configuration validation, and necessary builds locally; leave full automated regression to CI — tests that were not run locally must not be claimed as passing.
- Products, pages, and business processes are manually accepted by maintainers, who record the results.
- A local test database is not required to exist; database and Redis integration tests may only use explicitly isolated resources or CI service containers.
- Do not create phase/batch gates, long-running evidence, one-off reports, or `artifacts/`; clean up local Playwright/test artifacts after diagnostics are complete.

## Scenarios That Require Synchronized Documentation Updates

Whenever any of the following changes occur, the related documentation must be updated in sync:

| Change type | Must check / update |
| --- | --- |
| New or modified startup commands | `README.md`, `README.en.md`, `docs/development.md`, `package.json` script descriptions |
| New or modified Docker deployment | `README.md`, `README.en.md`, `docs/docker-deployment.md`, `.env.example` |
| New or modified environment variables | `.env.example`, `docs/env.md`, `docs/development.md`, `docs/docker-deployment.md` |
| New API or changed API contract | `docs/api.md`, frontend `services` / `types`, capability descriptions in the README |
| New Provider | `docs/provider.md`, `docs/provider-template.md`, README feature table, settings page description, example configuration |
| New backend page or route | README capability description, related `docs/`, menu / routing description |
| New async task or queue | `.env.example`, health check description, task center / worker documentation |
| New database table or key field | `docs/PROGRESS.md`, architecture / module documentation, migration notes and CI regression as needed |
| Changes to branching, CI, or PR process | `docs/branching.md`, `CONTRIBUTING.md`, PR template |
| Changes to security, secrets, or authorization logic | `SECURITY.md`, `.env.example`, related settings documentation |

## Configuration File Sync Rules

When configuration is involved, follow these rules:

1. When adding a new environment variable, update `.env.example` at the same time.
2. When Docker deployment also needs that variable, update both the single source of truth `.env.example` and `docker-compose.full.yml` at the same time.
3. When changing default ports, default paths, or default service names, update the README, development docs, and Docker docs at the same time.
4. When adding sensitive configuration, state whether it is encrypted at rest, whether it is masked when displayed, and whether it is prohibited from being written to logs.
5. When removing or renaming configuration, check scripts, CI, Docker, documentation, and the backend settings page.

Detailed environment variable descriptions are maintained in [env.md](env.md).

## Related-Content Check Rules

When an AI agent handles a task, it must first determine the type of change and check related content per [module-map.md](module-map.md):

1. Backend DTO / API changes: sync the frontend `services`, `types`, page fields, and [api.md](api.md).
2. Environment variable changes: sync the env template, Docker Compose, development / deployment docs, and [env.md](env.md).
3. Provider changes: sync settings, connection tests, masked display, [provider.md](provider.md), and [provider-template.md](provider-template.md).
4. CI / branching changes: sync the workflow, [branching.md](branching.md), `CONTRIBUTING.md`, and the PR template.
5. Open-source governance changes: sync the README, documentation center, `CHANGELOG.md`, and `.github/` configuration.

## AI Agent Workflow

When an AI agent modifies code, it should follow:

1. First rewrite the user's request into a short execution prompt: goal, task type, scope boundaries, required-reading entry points, fact confirmation, implementation strategy, verification, and knowledge capture.
2. Then form a minimal context package per [ai-workflow.md](ai-workflow.md): task goal, change type, related entry points, existing implementation, verification method, and risks.
3. Read files within a context budget: search and read locally first, then expand scope; do not stuff large files, large logs, or irrelevant context directly into the model.
4. Read the relevant code, configuration, and documentation first; do not assume scripts, ports, paths, or variables without verification.
5. Only modify files relevant to the task; do not opportunistically refactor unrelated modules.
6. Route business capability through the existing layering: handler → service → provider / repository / queue.
7. When AI, storage, images, platforms, or collection capabilities are involved, prefer extending via the Provider interface.
8. For time-consuming tasks, use task status and queues; do not block synchronously for a long time within a request.
9. When secrets are involved, route through encryption, masking, and log protection.
10. After completion, run the local checks matching the change per [task-checklist.md](task-checklist.md), and list the regression left to CI and the manual acceptance results.
11. For recurring issues, architectural decisions, tooling conventions, prompt templates, or quality gates, write them back into the corresponding document per [ai-workflow.md](ai-workflow.md) so subsequent AI tools can reuse them.

## Pre-Commit Checklist

- [ ] Code is consistent with the existing architecture.
- [ ] The user's request has been condensed into a short execution prompt, and relevant files have been read within the context budget.
- [ ] Context scope has been controlled per `docs/ai-workflow.md`, with necessary experience captured.
- [ ] No `.env` file, secret, token, cookie, or real platform credential has been committed.
- [ ] New / modified configuration has been synced to the single source of truth `.env.example` and related documentation.
- [ ] New / modified commands have been synced to the README and development documentation.
- [ ] New / modified Docker behavior has been synced to the Docker documentation.
- [ ] New / modified APIs, Providers, tasks, or pages have been synced to the related docs.
- [ ] Backend API / DTO changes have been synced to the frontend services / types and `docs/api.md`.
- [ ] Related files have been checked per `docs/module-map.md`.
- [ ] `go fmt ./...` has been run when backend Go code is involved.
- [ ] `pnpm build:admin` has been run or documented when admin is involved.
- [ ] `pnpm check:ui-copy --strict` has been run or documented when user-facing copy is involved.
- [ ] `pnpm build:collector` has been run or documented when the collector is involved.
- [ ] `docs/PROGRESS.md` has been updated for larger module or maintenance strategy changes.
