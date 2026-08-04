import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const P9_BATCH_6_JSON = 'docs/p9-task-batch-6-admin-inventory-center.json';
export const P9_BATCH_6_MD = 'docs/P9_TASK_BATCH_6_ADMIN_INVENTORY_CENTER.md';
export const P9_BATCH_6_GATE_JSON = 'docs/p9-task-batch-6-admin-inventory-center-gate.json';
export const P9_BATCH_6_GATE_MD = 'docs/P9_TASK_BATCH_6_ADMIN_INVENTORY_CENTER_GATE.md';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const TASK_IDS = ['P9-1001', 'P9-1002', 'P9-1003', 'P9-1004', 'P9-1005', 'P9-1006'];
const REQUIRED_FILES = [
  'admin/src/services/inventorySyncP9.ts',
  'admin/src/services/__tests__/inventorySyncP9.test.ts',
  'admin/src/pages/Ops/InventorySync/index.tsx',
  'admin/src/pages/Ops/InventorySync/RunDetail.tsx',
  'admin/src/pages/Ops/InventorySync/Calibration.tsx',
  'admin/src/pages/Ops/InventorySync/ManualBindings.tsx',
  'admin/src/pages/Ops/InventorySync/Bindings.tsx',
  'admin/src/pages/Ops/InventorySync/BindingDetail.tsx',
  'admin/src/pages/Ops/InventorySync/components/InventorySyncShared.tsx',
  'admin/e2e/mocks/inventory-sync-p9.ts',
  'admin/e2e/specs/p9-inventory-sync.spec.ts',
  'admin/config/routes.ts',
  P9_BATCH_6_MD,
  P9_BATCH_6_JSON,
];
const REQUIRED_ROUTES = [
  "/ops/inventory-sync'",
  "/ops/inventory-sync/runs/:runId'",
  "/ops/inventory-sync/calibration'",
  "/ops/inventory-sync/manual-bindings'",
  "/ops/inventory-sync/bindings'",
  "/ops/inventory-sync/bindings/:bindingId'",
];
const REQUIRED_API_PATHS = [
  "const BASE = '/api/v1/inventory-sync'",
  '${BASE}/runs',
  '${BASE}/runs/${enc(runId)}',
  '${BASE}/runs/${enc(runId)}/rerun',
  '${BASE}/runs/${enc(runId)}/snapshots',
  '${BASE}/runs/${enc(runId)}/audit-events',
  '${BASE}/snapshots/${enc(snapshotId)}',
  '${BASE}/snapshots/${enc(snapshotId)}/calibrations',
  '${BASE}/snapshots/${enc(snapshotId)}/recalibrate',
  '${BASE}/bindings',
  '${BASE}/bindings/${enc(bindingId)}',
  '${BASE}/bindings/${enc(bindingId)}/history',
  '${BASE}/manual-binding-requests',
  '${BASE}/manual-binding-requests/${enc(requestId)}',
  '${BASE}/manual-binding-requests/${enc(requestId)}/confirm',
  '${BASE}/manual-binding-requests/${enc(requestId)}/reject',
];
const FORBIDDEN_PATTERNS = [
  /\btenantId\b/,
  /\bpageSize\b/,
  /\btotalPages\b/,
  /\boffset\b/i,
  /access_token|refresh_token|client_secret|authorization:\s*bearer|cookie:/i,
];

function rootPath(rel) { return path.join(REPO_ROOT, rel); }
function read(rel) { try { return fs.readFileSync(rootPath(rel), 'utf8'); } catch { return ''; } }
function readJSON(rel) { try { return JSON.parse(read(rel)); } catch { return null; } }
function write(rel, value) {
  fs.mkdirSync(path.dirname(rootPath(rel)), { recursive: true });
  fs.writeFileSync(rootPath(rel), `${typeof value === 'string' ? value : JSON.stringify(value, null, 2)}\n`, 'utf8');
}
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}
function hasAll(text, values) { return values.every((value) => text.includes(value)); }
function noForbidden(text, patterns) { return patterns.every((pattern) => !pattern.test(text)); }

export function validateP9Batch6AdminInventoryCenterBundle({ evidence = {}, sources = {}, gitState = {} } = {}) {
  const service = sources.service ?? read('admin/src/services/inventorySyncP9.ts');
  const serviceTest = sources.serviceTest ?? read('admin/src/services/__tests__/inventorySyncP9.test.ts');
  const routes = sources.routes ?? read('admin/config/routes.ts');
  const pages = sources.pages ?? [
    'admin/src/pages/Ops/InventorySync/index.tsx',
    'admin/src/pages/Ops/InventorySync/RunDetail.tsx',
    'admin/src/pages/Ops/InventorySync/Calibration.tsx',
    'admin/src/pages/Ops/InventorySync/ManualBindings.tsx',
    'admin/src/pages/Ops/InventorySync/Bindings.tsx',
    'admin/src/pages/Ops/InventorySync/BindingDetail.tsx',
    'admin/src/pages/Ops/InventorySync/components/InventorySyncShared.tsx',
  ].map(read).join('\n');
  const e2e = sources.e2e ?? read('admin/e2e/specs/p9-inventory-sync.spec.ts');
  const mocks = sources.mocks ?? read('admin/e2e/mocks/inventory-sync-p9.ts') + read('admin/e2e/utils/routes.ts');
  const packageJSON = sources.packageJSON ?? read('package.json');
  const currentBranch = gitState.currentBranch ?? git(['branch', '--show-current']);
  const currentHead = gitState.currentHead ?? git(['rev-parse', 'HEAD']);
  const stagedFileCount = gitState.stagedFileCount ?? (git(['diff', '--cached', '--name-only']) ? git(['diff', '--cached', '--name-only']).split('\n').filter(Boolean).length : 0);
  const dirty = gitState.workingTreeDirty ?? Boolean(git(['status', '--short']));
  const adminText = [service, serviceTest, routes, pages, e2e, mocks].join('\n');
  const contractText = [service, pages, e2e].join('\n');
  const checks = [
    ['requiredFilesPresent', REQUIRED_FILES.every((rel) => fs.existsSync(rootPath(rel)))],
    ['batchId', evidence.batchId === 'P9-TASK-BATCH-6'],
    ['currentBranch', currentBranch === 'dev'],
    ['currentHeadPresent', typeof currentHead === 'string' && currentHead.length > 0],
    ['stagedFileCount', stagedFileCount === 0],
    ['workingTreeDirtyRecorded', evidence.workingTreeDirty === dirty],
    ['changesCommitted', evidence.changesCommitted === false],
    ...TASK_IDS.map((id) => [`${id} status`, evidence.tasks?.[id]?.status === 'completed']),
    ['adminUiImplemented', evidence.adminUiImplemented === true],
    ['routesPresent', hasAll(routes, REQUIRED_ROUTES)],
    ['serviceRoutesExact', hasAll(service, REQUIRED_API_PATHS)],
    ['cursorPaginationOnly', evidence.keysetPagination === true && evidence.offsetPaginationAbsent === true && hasAll(service, ['items: T[]', 'nextCursor', 'hasMore', 'limit']) && noForbidden(service, [/pageSize/, /totalPages/, /offset/i])],
    ['writeBodiesExact', evidence.writeBodiesExact === true && hasAll(service + serviceTest, [
      'shopConnectionId: string',
      'platform: string',
      'providerMode: string',
      'fixtureScenario?: string',
      'expectedRevision: number',
      'expectedCalibrationVersion: number',
      'selectedLocalSkuId: string',
      'reasonCode: string',
    ])],
    ['idempotencyKeys', evidence.writesRequireIdempotencyKey === true && hasAll(service + serviceTest, ['Idempotency-Key', 'key-create', 'key-rerun', 'key-recalibrate', 'key-confirm', 'key-reject'])],
    ['skuSearchContract', evidence.productSkuSearchReused === true && pages.includes("searchProductSkus({ keyword: text, limit: 10 })") && !pages.includes('tenantId') && read('admin/src/services/products.ts').includes('/api/v1/product-skus/search')],
    ['allowedActionsOnly', evidence.allowedActionsOnly === true && hasAll(pages, ['canViewSnapshots', 'canRerun', 'canViewAudit', 'canViewHistory', 'canConfirm', 'canReject'])],
    ['nonProductionBoundary', evidence.fixtureBoundaryVisible === true && hasAll(pages, ['realPlatformNetworkCalls=0', 'inventoryMutationCalls=0', 'productionReady=false', 'p9Complete=false'])],
    ['safeAuditMetadata', evidence.rawAuditMetadataExposed === false && hasAll(pages, ['safeAuditSummary', 'AuditMetadataBlock'])],
    ['e2eCoverage', evidence.e2eCoveragePassed === true && hasAll(e2e, ['viewports', 'expectNoRootOverflow', 'writeGuard.allow', 'create-run'])],
    ['mocksRegistered', mocks.includes('inventorySyncP9Response(path)') && mocks.includes('/api/v1/inventory-sync/runs')],
    ['adminForbiddenContext', noForbidden(contractText, FORBIDDEN_PATTERNS)],
    ['p10BoundaryPreserved', evidence.p10BoundaryPreserved === true
      && evidence.productionReady === false
      && evidence.p9Complete === false
      && evidence.realDouyinProviderImplemented === false
      && evidence.oauthImplemented === false
      && evidence.realPlatformReadEnabled === false
      && evidence.realPlatformWriteEnabled === false
      && evidence.workerImplemented === false
      && evidence.backgroundSyncWorkerImplemented === false
      && evidence.automaticRetryWorkerImplemented === false],
    ['packageScripts', hasAll(packageJSON, ['test:p9-task-batch-6', 'p9:task-batch-6-gate'])],
    ['docsProgress', evidence.docsUpdated === true && read('docs/P9_EXECUTION_PLAN.md').includes('P9-1006') && read('docs/PROGRESS.md').includes('Batch 6')],
  ];
  const failed = checks.filter(([, ok]) => !ok).map(([id]) => id);
  return {
    status: failed.length === 0 ? 'passed' : 'failed',
    failed,
    failedCount: failed.length,
    currentBranch,
    currentHead,
    stagedFileCount,
    workingTreeDirty: dirty,
    requiredFiles: REQUIRED_FILES,
    checks: checks.map(([id, ok]) => ({ id, status: ok ? 'passed' : 'failed' })),
  };
}

export function renderP9Batch6AdminInventoryCenterGateMarkdown(report) {
  return `# P9 Task Batch 6 Admin Inventory Center Gate

Status: **${report.status}**

- Current branch: ${report.currentBranch}
- Current HEAD: ${report.currentHead}
- Staged files: ${report.stagedFileCount}
- Working tree dirty: ${report.workingTreeDirty}
- Failed checks: ${report.failedCount ? report.failed.join(', ') : 'none'}

${report.checks.map((check) => `- ${check.status === 'passed' ? 'PASS' : 'FAIL'} \`${check.id}\``).join('\n')}
`;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  const evidence = readJSON(P9_BATCH_6_JSON) ?? {};
  const report = validateP9Batch6AdminInventoryCenterBundle({ evidence });
  write(P9_BATCH_6_GATE_JSON, report);
  write(P9_BATCH_6_GATE_MD, renderP9Batch6AdminInventoryCenterGateMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.status !== 'passed') process.exitCode = 1;
}
