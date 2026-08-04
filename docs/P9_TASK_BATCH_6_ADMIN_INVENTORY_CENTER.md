# P9 Task Batch 6 Admin Inventory Sync and Binding Center

Status: **Completed Locally**

```text
phase=P9
batchId=P9-TASK-BATCH-6
batchName=Admin Inventory Sync and Binding Center
completedTaskIds=P9-1001,P9-1002,P9-1003,P9-1004,P9-1005,P9-1006
workingBranch=dev
committed=false
productionReady=false
p9Complete=false
p10BoundaryPreserved=true
```

## Scope

Batch 6 adds the Admin UI and frontend client for the fixture-only P9 inventory sync API delivered in Batch 5.

- `P9-1001` Inventory Sync Dashboard: `/ops/inventory-sync`
- `P9-1002` Sync Run Detail: `/ops/inventory-sync/runs/:runId`
- `P9-1003` SKU Calibration Workspace: `/ops/inventory-sync/calibration`
- `P9-1004` Manual Binding Workspace: `/ops/inventory-sync/manual-bindings`
- `P9-1005` Binding History and Audit: `/ops/inventory-sync/bindings` and `/ops/inventory-sync/bindings/:bindingId`
- `P9-1006` Admin UX Verification: fixture copy, loading/error/empty states, cursor pagination, write guards, and responsive E2E coverage

## API Binding

The Admin client uses only the real Batch 5 API contract under `/api/v1/inventory-sync`.

- List endpoints use `{items,nextCursor,hasMore,limit}` and keyset cursor parameters only.
- No page/offset/total pagination is fabricated.
- Product SKU search reuses `GET /api/v1/product-skus/search` with only `keyword`, `productId`, and `limit`.
- The Admin UI does not send caller-supplied `tenantId`, actor, role, credential, OAuth, or platform credential fields.
- Writes use hidden `Idempotency-Key` headers and exact bodies:
  - `POST /runs`: `{shopConnectionId, platform, providerMode, fixtureScenario?}`
  - `POST /runs/:runId/rerun`: `{expectedRevision}`
  - `POST /snapshots/:snapshotId/recalibrate`: `{expectedCalibrationVersion, reason}`
  - `POST /manual-binding-requests/:requestId/confirm`: `{expectedRevision, selectedLocalSkuId, comment?}`
  - `POST /manual-binding-requests/:requestId/reject`: `{expectedRevision, reasonCode, comment?}`

## Safety Boundary

Every new P9 Admin page shows the fixture/mock boundary:

```text
realPlatformNetworkCalls=0
inventoryMutationCalls=0
realDouyinProviderImplemented=false
oauthImplemented=false
realPlatformReadEnabled=false
realPlatformWriteEnabled=false
workerImplemented=false
backgroundSyncWorkerImplemented=false
automaticRetryWorkerImplemented=false
productionReady=false
p9Complete=false
```

`succeeded` run status is rendered as fixture/test completion, not real inventory synchronization.

## Verification

Executed during implementation:

```text
pnpm --filter @trademind/admin test -- inventorySyncP9 = passed
pnpm --filter @trademind/admin exec playwright test --config ../playwright.config.ts --grep @p9-inventory-sync = passed
pnpm build:admin = passed
pnpm test:p9-task-batch-6 = passed
pnpm p9:task-batch-6-gate = passed
```

Additional gate commands are recorded in `P9_TASK_BATCH_6_ADMIN_INVENTORY_CENTER_GATE.md`.
