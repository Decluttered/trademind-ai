# Douyin Shop Manual Acceptance Pre-Check Guide

> **Douyin Shop Release Candidate** · **Douyin Production Adapter Implemented** · Real platform acceptance must be performed manually in an authorized environment · Not Production Ready

## Scope

- The current adapter code covers OAuth, tokens, categories, images, platform drafts, orders, inventory, webhook signatures, and event routing
- Repository scripts and CI **do not execute** real Douyin Shop write paths, do not auto-write to irreversible endpoints, and never list products directly
- Missing-credentials state: `blocked_by_real_credentials` / `environment_required` (**not** a system P0/P1 failure)
- Creating a local draft ≠ a real platform draft; creating a Douyin Shop draft ≠ the product being listed
- **Code implementation complete ≠ real E2E has passed**

## Prerequisite Checklist

| Check | Pass Criteria | State When Missing |
| --- | --- | --- |
| App Key / App Secret | Configured (shown masked) | Awaiting real credentials |
| OAuth authorization | >=1 authorized shop | Not authorized |
| Token validity | Structural check / optional live test | Requires manual confirmation |
| Category / attribute sync | Cached data present | Currently skipped |
| Storage `public_base` | Publicly reachable via HTTPS | Awaiting public-facing Storage |
| Douyin Shop image upload | Pre-check passes | Awaiting public-facing Storage |
| Release Candidate label | Retained | — |

## Page Entry Points

| Page | Purpose |
| --- | --- |
| `/settings/platforms?platform=douyin_shop` | Douyin Shop integration + production pre-check panel |
| `/settings/config-status` | Overview of credentials / Storage / publish capability |
| `/product/drafts` | Product preparation before publishing |
| `/product/publish-batches` | Batch publish scope notices |
| `/ops/task-center/failures?platform=douyin_shop` | Douyin Shop-related failures |

## Production Maintenance Acceptance Process

- Automated regression runs in GitHub Actions in an isolated environment.
- Maintainers follow [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md) to complete page, read-only API, and controlled business-flow checks.
- Real platform write operations require external production approval beforehand; if credentials or the environment are missing, record `blocked_by_real_credentials` / `blocked_by_environment`.
- Conclusions are recorded in the PR or release ticket; do not commit one-off JSON, Markdown reports, screenshots, or logs.

## Missing-Credentials Message (Standard Copy)

> Real Douyin Shop credentials are not currently configured. The system can only run local demos and pre-checks — it cannot execute a real Douyin Shop E2E run.

Buttons: Configure Platform Credentials · View E2E Readiness Checklist · View Failed Tasks

## Related Documents

- [`DOUYIN_E2E_CHECKLIST.md`](DOUYIN_E2E_CHECKLIST.md)
- [`PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md`](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md)
- [`STORAGE_PUBLIC_URL_GUIDE.md`](STORAGE_PUBLIC_URL_GUIDE.md)
