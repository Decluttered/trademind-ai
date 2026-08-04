import assert from 'node:assert/strict';
import { validateP9Batch6AdminInventoryCenterBundle } from '../../../scripts/p9-task-batch-6-admin-inventory-center-gate.mjs';

const tasks = Object.fromEntries(['P9-1001', 'P9-1002', 'P9-1003', 'P9-1004', 'P9-1005', 'P9-1006'].map((id) => [id, { status: 'completed' }]));
const evidence = {
  batchId: 'P9-TASK-BATCH-6',
  changesCommitted: false,
  workingTreeDirty: true,
  tasks,
  adminUiImplemented: true,
  keysetPagination: true,
  offsetPaginationAbsent: true,
  writeBodiesExact: true,
  writesRequireIdempotencyKey: true,
  productSkuSearchReused: true,
  allowedActionsOnly: true,
  fixtureBoundaryVisible: true,
  rawAuditMetadataExposed: false,
  e2eCoveragePassed: true,
  docsUpdated: true,
  p10BoundaryPreserved: true,
  productionReady: false,
  p9Complete: false,
  realDouyinProviderImplemented: false,
  oauthImplemented: false,
  realPlatformReadEnabled: false,
  realPlatformWriteEnabled: false,
  workerImplemented: false,
  backgroundSyncWorkerImplemented: false,
  automaticRetryWorkerImplemented: false,
};

const sources = {
  service: `
type CursorPage<T> = { items: T[]; nextCursor?: string; hasMore: boolean; limit: number };
const BASE = '/api/v1/inventory-sync';
postJSON(BASE + '/runs', body, { headers: { 'Idempotency-Key': key } });
\`\${BASE}/runs\`;
\`\${BASE}/runs/\${enc(runId)}\`;
\`\${BASE}/runs/\${enc(runId)}/rerun\`;
\`\${BASE}/runs/\${enc(runId)}/snapshots\`;
\`\${BASE}/runs/\${enc(runId)}/audit-events\`;
\`\${BASE}/snapshots/\${enc(snapshotId)}\`;
\`\${BASE}/snapshots/\${enc(snapshotId)}/calibrations\`;
\`\${BASE}/snapshots/\${enc(snapshotId)}/recalibrate\`;
\`\${BASE}/bindings\`;
\`\${BASE}/bindings/\${enc(bindingId)}\`;
\`\${BASE}/bindings/\${enc(bindingId)}/history\`;
\`\${BASE}/manual-binding-requests\`;
\`\${BASE}/manual-binding-requests/\${enc(requestId)}\`;
\`\${BASE}/manual-binding-requests/\${enc(requestId)}/confirm\`;
\`\${BASE}/manual-binding-requests/\${enc(requestId)}/reject\`;
shopConnectionId: string; platform: string; providerMode: string; fixtureScenario?: string;
expectedRevision: number; expectedCalibrationVersion: number; selectedLocalSkuId: string; reasonCode: string;
Idempotency-Key key-create key-rerun key-recalibrate key-confirm key-reject
`,
  serviceTest: 'Idempotency-Key key-create key-rerun key-recalibrate key-confirm key-reject',
  routes: `
'/ops/inventory-sync'
'/ops/inventory-sync/runs/:runId'
'/ops/inventory-sync/calibration'
'/ops/inventory-sync/manual-bindings'
'/ops/inventory-sync/bindings'
'/ops/inventory-sync/bindings/:bindingId'
`,
  pages: `
searchProductSkus({ keyword: text, limit: 10 })
canViewSnapshots canRerun canViewAudit canViewHistory canConfirm canReject
realPlatformNetworkCalls=0 inventoryMutationCalls=0 productionReady=false p9Complete=false
safeAuditSummary AuditMetadataBlock
`,
  e2e: 'viewports expectNoRootOverflow writeGuard.allow create-run',
  mocks: 'inventorySyncP9Response(path) /api/v1/inventory-sync/runs',
  packageJSON: 'test:p9-task-batch-6 p9:task-batch-6-gate',
};

function validate(overrides = {}, sourceOverrides = {}, gitOverrides = {}) {
  return validateP9Batch6AdminInventoryCenterBundle({
    evidence: { ...evidence, ...overrides },
    sources: { ...sources, ...sourceOverrides },
    gitState: {
      currentBranch: 'dev',
      currentHead: 'abc123',
      stagedFileCount: 0,
      workingTreeDirty: true,
      ...gitOverrides,
    },
  });
}

function expectFailed(check, overrides = {}, sourceOverrides = {}, gitOverrides = {}) {
  const result = validate(overrides, sourceOverrides, gitOverrides);
  assert.equal(result.status, 'failed');
  assert.ok(result.failed.includes(check), `${check} should fail; actual failures: ${result.failed.join(', ')}`);
}

const valid = validate();
assert.equal(valid.status, 'passed');

expectFailed('P9-1004 status', { tasks: { ...tasks, 'P9-1004': { status: 'planned' } } });
expectFailed('cursorPaginationOnly', { offsetPaginationAbsent: false }, { service: `${sources.service}\npageSize totalPages offset` });
expectFailed('writeBodiesExact', { writeBodiesExact: false });
expectFailed('idempotencyKeys', { writesRequireIdempotencyKey: false });
expectFailed('skuSearchContract', { productSkuSearchReused: false });
expectFailed('allowedActionsOnly', { allowedActionsOnly: false }, { pages: sources.pages.replace('canConfirm', '') });
expectFailed('nonProductionBoundary', { fixtureBoundaryVisible: false });
expectFailed('safeAuditMetadata', { rawAuditMetadataExposed: true });
expectFailed('e2eCoverage', { e2eCoveragePassed: false });
expectFailed('adminForbiddenContext', {}, { pages: `${sources.pages}\ntenantId` });
expectFailed('p10BoundaryPreserved', { productionReady: true, realPlatformWriteEnabled: true });
expectFailed('stagedFileCount', {}, {}, { stagedFileCount: 1 });
expectFailed('changesCommitted', { changesCommitted: true });

console.log('p9 task batch 6 admin inventory center fixtures passed');
