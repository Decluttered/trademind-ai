# P10 Manual Acceptance Checklist

Status: **Prepared / Pending Manual Acceptance**

This checklist validates repository-side P10 development without contacting Douyin or promoting runtime beyond L0. Record the operator, date, environment, source HEAD, tenant, fixture shop, observed result and redacted evidence for every item. Do not record credential values, OAuth state values, Authorization/Cookie headers, database/Redis URLs or raw Provider responses.

Automated tests, Gates, PostgreSQL runtime, Playwright, Vitest, Jest, `go test` and `go test -race` were deferred by the Owner for this development round. No item below is pre-marked as passed.

## A. Credential

- [ ] Create a development-only offline credential for an authorized fixture shop; confirm the response contains metadata only.
- [ ] List credentials as the same tenant and verify credential ID, platform, shop, status, expiry, algorithm and revision.
- [ ] Attempt cross-tenant lookup/use and confirm denial without existence leakage.
- [ ] Rotate with the current revision; confirm a new active encrypted version and retired prior version.
- [ ] Retry rotation with a stale revision and confirm conflict.
- [ ] Revoke locally; confirm `status=revoked` and all later decrypt/use attempts are denied.
- [ ] Advance/use an expired fixture credential and confirm `status=expired` or credential-use denial.
- [ ] Inspect Admin, API, logs and audit evidence for absence of Token, ciphertext and secrets.

## B. Offline OAuth Foundation

- [ ] Start authorization using an exact allowlisted redirect and confirm `networkRequestExecuted=false`.
- [ ] Confirm the persisted state is random, hashed, expiring and bound to tenant, user, platform, shop and redirect.
- [ ] Complete the fixture callback once and confirm safe credential metadata creation.
- [ ] Replay the same state and confirm rejection.
- [ ] Use an expired state and confirm rejection.
- [ ] Use the state from another tenant/user and confirm rejection.
- [ ] Use a redirect not in the exact allowlist and confirm rejection.
- [ ] Confirm `oauth_authorization_started`, `oauth_callback_received` and credential lifecycle audit events contain no secrets.

## C. Provider

- [ ] Confirm `DouyinReadOnlyInventoryProvider` conforms to the exported P9 `InventoryProvider` interface.
- [ ] Confirm local publication pagination uses only the repository-confirmed `product.detail` operation.
- [ ] Confirm no all-shop endpoint was invented and no P10 write method/route is reachable.
- [ ] Review bounded connection/request/header timeouts, connection pool, page limit, <=100 SKU limit and response-size limit.
- [ ] Exercise offline/fake mappings for unauthorized, expired, rate-limited, unavailable, invalid-request and protocol errors.
- [ ] Confirm 429 exposes safe rate-limit state and `Retry-After` metadata without automatic business retry.
- [ ] Confirm internal and Provider request IDs correlate safe status/snapshot/audit evidence.

## D. Inventory Sync

- [ ] From an authorized tenant/shop fixture, manually trigger one read-only run with a unique `Idempotency-Key`.
- [ ] Confirm immutable snapshots, normalization and <=100 total SKU enforcement.
- [ ] Confirm existing P9 SKU binding calibration and manual-binding fallback are reused.
- [ ] Confirm run history, snapshot counts, calibration counts, manual backlog and audit correlation.
- [ ] Repeat the same idempotency key/payload and confirm the original result is returned.
- [ ] Repeat the key with a different payload and confirm conflict.
- [ ] Manually rerun only a failed/cancelled tenant-scoped source revision; confirm no automatic retry.

## E. Admin

- [ ] Open `/ops/p10-readiness` and verify environment, L0 boundary, Provider, credential metadata and read-only status.
- [ ] Verify tenant/shop scope, last sync, last safe error and rate-limit state.
- [ ] Verify all five kill switches; Write remains permanently blocked.
- [ ] Save allowlist and Gray draft with revision checks; verify Owner/Technical Lead approvals reset and cannot be self-generated.
- [ ] Verify pause and stop immediately produce a blocked Gray state.
- [ ] Verify Admin never displays access token, refresh token, ciphertext, app secret or client secret.
- [ ] Verify `p10.read`, `p10.credential.manage` and `p10.control.manage` RBAC for admin/operator/reviewer/readonly.
- [ ] Manually inspect normal/loading/empty/error/readonly/disabled/submitting states.
- [ ] Inspect 1440x900, 1280x800, 1024x768, 768x900 and 375x812; verify root horizontal overflow is absent and tables scroll only internally.
- [ ] Intercept every non-GET request during UI acceptance; confirm cancel=0, confirm=1, rapid repeat=1 and no extra write request.

## F. Security

- [ ] Confirm tenant/store authorization on credential, control and read-run routes.
- [ ] Confirm strict JSON rejects unknown fields, multiple values and oversized bodies.
- [ ] Confirm Provider base URL is trusted config, official HTTPS host only, with no userinfo/query/fragment or non-443 port.
- [ ] Confirm OAuth replay, redirect allowlist and state binding fail closed.
- [ ] Confirm redaction covers Authorization, Cookie, access/refresh token, app/client secret, database URL and Redis URL keys.
- [ ] Confirm revoked/expired credentials cannot decrypt or invoke a Provider.
- [ ] Confirm kill switches override feature flags and all real calls remain denied at L0.
- [ ] Confirm no P10 inventory write capability, Worker, scheduler, queue consumer or automatic business retry exists.

## G. Gray

- [ ] Configure at most one tenant, one shop and `maxSku<=100`; confirm overflow is rejected.
- [ ] Confirm Draft, PendingApproval, Approved, Active, Paused and Stopped are represented by the model.
- [ ] Confirm Approved/Active evaluation requires both Owner and Technical Lead approval.
- [ ] Confirm the system exposes no API that automatically grants either human approval.
- [ ] Confirm Pause and Stop block Provider read through the kill-switch/Gray guard.
- [ ] Confirm runtime remains L0 and no real Gray begins during this checklist.

## H. Recovery and Operations

- [ ] Review `backup-preproduction.sh` artifact, checksum, metadata, retention configuration and environment guard.
- [ ] Review `restore-preproduction.sh` explicit isolated target identity and production-restore denial.
- [ ] Review `rollback-preproduction.sh` previous immutable image selection, readiness check and non-implicit database restore.
- [ ] Execute timed backup/restore/rollback and five kill-switch drills only after independent pre-production exists.
- [ ] Record measured RPO/RTO, alert fire/recovery and dedicated-host performance evidence; do not infer these from code/build checks.

## Real Platform Acceptance

Status: **blocked_by_external_infrastructure_and_credentials**

The following remain separate and must not be attempted until an independent pre-production server, managed key source, Douyin app credential and revocable test/allowlisted shop are available:

- [ ] Real OAuth authorization and callback.
- [ ] Real credential rotation/revocation drill.
- [ ] Real `product.detail` read, pagination, timeout and rate-limit validation.
- [ ] Real read integration, snapshot/calibration/manual rerun evidence.
- [ ] G0/G1 read-only Gray with Owner + Technical Lead approval.
- [ ] Dedicated-host performance, soak, alerts, RPO/RTO and final Production Acceptance.

Production inventory write is not part of this checklist and remains unapproved.
