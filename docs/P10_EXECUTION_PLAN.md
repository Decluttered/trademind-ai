# P10 Execution Plan

Status: **Repository-side Development Complete / Manual Acceptance Ready / Runtime L0**

The Owner-approved plan contains 11 workstreams. Repository-side foundation/development for `P10-W2` through `P10-W9` is complete, and `P10-W10` has a manual-acceptance package for this round. Automated verification, external provisioning/activation, real read/gray, performance acceptance, final Production Acceptance, Tag and Release remain deferred. `P10-W11` is conditional, deferred pending separate Owner approval, and not required for initial Production Ready.

The authoritative machine-readable plan is [`p10-execution-plan.json`](p10-execution-plan.json). The original draft remains unchanged in [`P10_EXECUTION_PLAN_DRAFT.md`](P10_EXECUTION_PLAN_DRAFT.md) and [`p10-execution-plan-draft.json`](p10-execution-plan-draft.json).

## Current Development and Acceptance Status

| Batch | Development | Verification | Manual acceptance | External activation |
| --- | --- | --- | --- | --- |
| P10-B1 | Repository foundation completed | `deferred_by_owner_for_manual_acceptance` | pending | external pre-production provisioning deferred; batch not fully complete |
| P10-B2 | completed | `deferred_by_owner_for_manual_acceptance` | pending | real OAuth deferred |
| P10-B3 | completed | `deferred_by_owner_for_manual_acceptance` | pending | real Provider activation deferred |
| P10-B4 | completed | `deferred_by_owner_for_manual_acceptance` | pending | real inventory read activation deferred |
| P10-B5 | completed | `deferred_by_owner_for_manual_acceptance` | pending | recovery drills deferred |
| P10-B6 | completed | `deferred_by_owner_for_manual_acceptance` | pending | security acceptance deferred |
| P10-B7 | completed | `deferred_by_owner_for_manual_acceptance` | pending | dedicated-host performance acceptance deferred |
| P10-B8 | completed | `deferred_by_owner_for_manual_acceptance` | pending | real Gray deferred |
| P10-B9 | manual acceptance package prepared | `deferred_by_owner_for_manual_acceptance` | pending execution | Production Acceptance deferred |

This status records development only. It does not satisfy any gate or acceptance criterion and does not authorize promotion beyond `currentAllowedLevel=L0`.

## Batch 1 - Pre-production Foundation

- Task IDs: P10-101 to P10-104.
- Entry: Owner gate passed, boundary approved at L0, P9 transition reusable. Dependencies: W1 and P9 closure.
- Scope: isolated pre-production topology, PostgreSQL, Redis, deployment/domain/config/secrets boundary, observability and fail-closed environment guards.
- Forbidden: OAuth, real credentials, Provider/network/read/write, Worker/retry/gray.
- Evidence/tests/gate: isolation and topology evidence, config inventory without values, teardown rehearsal, isolation/denial/config/quality/architecture tests, Batch 1 gate.
- Exit/rollback/approval: resources are independently isolated and repeatably removable; disable deployment and revoke test-only material on failure. No new Owner decision, and production resource use remains prohibited.

## Batch 2 - Credential Security and OAuth Foundation

- Task IDs: P10-201 to P10-204. Entry: Batch 1 plus reviewed credential threat model. Dependencies: managed key/KMS and existing settings/audit.
- Scope: backend-only OAuth, encrypted envelope storage, rotation, revocation, and redacted audit. Forbidden: frontend token storage, production credentials, real Provider calls, inventory access, gray.
- Evidence/tests/gate: encryption/key separation, rotation/revocation drills, OAuth replay/tenant tests, secret scans, Batch 2 gate.
- Exit/rollback/approval: no credential exposure and all lifecycle drills pass; disable routes and revoke test credentials on failure. Production credential activation waits for Gray dual approval.

## Batch 3 - Real Douyin Read-only Provider

- Task IDs: P10-301 to P10-304. Entry: Batch 2 and frozen read-only contract. Dependencies: `InventoryProviderPort` and revocable test-shop access.
- Scope: read-only adapter, bounded GET client, rate-limit/error mapping, normalization. Forbidden: second domain, mutation, publish/listing/write, business auto-retry, production shop.
- Evidence/tests/gate: port conformance, endpoint allowlist, redacted read evidence, fake/mock failure tests, no-write reachability, Batch 3 gate.
- Exit/rollback/approval: only approved reads are reachable; activate Provider kill switch, revoke credential, and return to fixture/mock on failure. D01/D02 cover implementation only.

## Batch 4 - Real Read Integration and Safety Controls

- Task IDs: P10-401 to P10-404. Entry: Batch 3 and operational Provider kill switch. Dependencies: existing inventory sync plus tenant/shop authorization.
- Scope: manual read integration, tenant/shop allowlists, 100-SKU guard, snapshot/audit/manual rerun. Forbidden: production gray, expanded scope, writes, Worker, auto-retry.
- Evidence/tests/gate: golden path, fail-closed scope cases, audit correlation, boundary/write-denial/idempotent-read/provider-failure tests, Batch 4 gate.
- Exit/rollback/approval: scoped reads pass while mutation stays disabled; disable read/Provider paths and return to fixture/mock on failure. Scope expansion requires a new dual approval.

## Batch 5 - Backup, Restore, Rollback and Kill Switch

- Task IDs: P10-501 to P10-505. Entry: Batch 4 and frozen RPO/RTO. Dependencies: pre-production and operations runbooks.
- Scope: backup/PITR, timed restore, deployment/migration rollback, credential revocation, and provider/tenant/shop/read/write kill switches. Forbidden: destructive production drills, writes, unmeasured claims.
- Evidence/tests/gate: measured RPO/RTO, restore consistency, rollback timings, five drill records and Batch 5 gate.
- Exit/rollback/approval: RPO <=15m, RTO <=60m and every rollback/kill switch passes; stop, restore, revoke, and keep Provider disabled on uncertainty. Owner reviews evidence before Gray.

## Batch 6 - Security Hardening and Adversarial Verification

- Task IDs: P10-601 to P10-604. Entry: Batch 5 with isolated security environment. Dependencies: threat model and scanning toolchain.
- Scope: authorization/tenant attacks, OAuth replay/SSRF/injection, secret/dependency/container scans, audit/privilege checks. Forbidden: production testing, secret recording, broad waivers, capability enablement.
- Evidence/tests/gate: threat disposition, redacted reports, adversarial closure, auth/OAuth/security/sensitive suites and Batch 6 gate.
- Exit/rollback/approval: no open Critical/High finding and isolation remains fail closed; disable paths, revoke credentials, and restore last verified deployment. Operations/Security sign-off is recorded for final acceptance.

## Batch 7 - Capacity, Performance and Observability

- Task IDs: P10-701 to P10-705. Entry: Batch 6 and an exclusive dedicated host. Dependencies: P7 evidence and real target specifications.
- Scope: fresh B1-C1-C2-B2, at least three repeats, complete latency/throughput/error/resource/provider-limit evidence, long-run stability, dashboards/alerts, and SLO freeze before G1.
- Forbidden: accepting the P7 waiver, noisy hosts, unfrozen G1 SLOs, gray. Evidence/tests/gate include host identity, raw runs, comparability/stability/alert checks, architecture loader repair, and Batch 7 gate.
- Exit/rollback/approval: repeatable evidence meets frozen SLOs and G1 telemetry is ready; abort promotion and disable reads on instability. Owner reviews the frozen SLOs before dual Gray approval.

## Batch 8 - Read-only Gray Release

- Task IDs: P10-801 to P10-804. Entry: Batch 7, all blocking G1 criteria, and Owner plus Technical Lead Go approval.
- Scope: G0 then G1, one tenant, one shop, <=100 SKUs, manual trigger/rerun, observation/audit/rollback. Forbidden: expanded scope, GA, mutation, Worker, auto-retry.
- Evidence/tests/gate: dual approval, scope snapshot, observations/incidents, runtime boundary/write-denial/kill-switch/alert checks and Batch 8 Go/No-Go gate.
- Exit/rollback/approval: read-only gray passes without unauthorized writes; disable read/Provider, revoke the shop credential, and return to the prior level on failure. Both human approvals are mandatory.

## Batch 9 - Final Production Acceptance and Release (Current Round: Manual Acceptance Preparation Only)

- Task IDs: P10-901 to P10-904. Entry: Batch 8, complete initial criteria, technical and operations/security sign-offs.
- Scope: evidence aggregation, release integrity, final production gate, Owner final decision. Forbidden: automatic approval, requiring real write, early release, silent baseline waiver.
- Evidence/tests/gate: final matrix and sign-offs, full required regression, contract/architecture/security gates, and Batch 9 final gate.
- Exit/rollback/approval: `productionReady=true` is allowed only after the gate and explicit Owner approval; otherwise no release, disable Provider access, and retain the prior accepted release.

## Conditional Write Extension

`P10-W11` remains `deferred_pending_separate_owner_approval`, `conditional=true`, and `requiredForInitialProductionReady=false`. It cannot start until read-only gray passes and the Owner separately approves writes. Automatic, bulk, and silently retried writes remain forbidden.

Next action: **Manual Acceptance**. `p10RepositoryDevelopmentComplete=true`, `p10ManualAcceptanceReady=true`, `productionReady=false`, and `productionAcceptancePassed=false`.
