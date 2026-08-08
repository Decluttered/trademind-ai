import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  computeLiveProtectedSourceManifest,
  readProtectedSourceFreeze,
  validateProtectedSourceFreezeBundle,
} from './p9-protected-source-freeze.mjs';

export const P9_BATCH_7_JSON = 'docs/p9-task-batch-7-integration-development-closure.json';
export const P9_BATCH_7_MD = 'docs/P9_TASK_BATCH_7_INTEGRATION_DEVELOPMENT_CLOSURE.md';
export const P9_BATCH_7_GATE_JSON = 'docs/p9-task-batch-7-integration-development-closure-gate.json';
export const P9_BATCH_7_GATE_MD = 'docs/P9_TASK_BATCH_7_INTEGRATION_DEVELOPMENT_CLOSURE_GATE.md';
export const P9_BATCH_7_RUNTIME_JSON = 'artifacts/p9-batch7-runtime.json';
export const P9_BATCH_7_RUNTIME_JSONL = 'artifacts/p9-batch7-runtime.jsonl';
export const P9_BATCH_7_E2E_JSONL = 'artifacts/p9-batch7-e2e.jsonl';
export const P9_BATCH_7_RACE_JSONL = 'artifacts/p9-batch7-race.jsonl';
export const P9_BATCH_7_SOURCE_MANIFEST_JSON = 'artifacts/p9-batch7-source-manifest.json';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const TASK_IDS = ['P9-1101', 'P9-1102', 'P9-1103', 'P9-1104', 'P9-1105'];
const ACCEPTANCE_IDS = ['AC-P9-09', 'AC-P9-10', 'AC-P9-11', 'AC-P9-12', 'AC-P9-13', 'AC-P9-14', 'AC-P9-15'];
const REQUIRED_FILES = [
  P9_BATCH_7_JSON,
  P9_BATCH_7_MD,
  P9_BATCH_7_RUNTIME_JSON,
  P9_BATCH_7_RUNTIME_JSONL,
  P9_BATCH_7_E2E_JSONL,
  P9_BATCH_7_RACE_JSONL,
  P9_BATCH_7_SOURCE_MANIFEST_JSON,
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  'artifacts/p9-postgres-runtime.json',
  'docs/P9_POSTGRESQL_INTEGRATION_CLOSURE.md',
  'docs/p9-postgresql-integration-closure.json',
  'docs/p9-postgresql-integration-closure-gate.json',
  'docs/P9_TASK_BATCH_6_ADMIN_INVENTORY_CENTER.md',
  'docs/p9-task-batch-6-admin-inventory-center.json',
  'docs/p9-task-batch-6-admin-inventory-center-gate.json',
  'admin/e2e/specs/p9-inventory-sync.spec.ts',
  'admin/e2e/mocks/inventory-sync-p9.ts',
  'admin/src/services/inventorySyncP9.ts',
  'backend/internal/testing/integration/p9_postgres_integration_test.go',
  'backend/internal/modules/inventorysyncp9/postgres_integration_test.go',
  'backend/internal/modules/inventorysyncp9/postgres_contract_test.go',
  'scripts/p9-task-batch-7-runtime.mjs',
  'scripts/p9-task-batch-7-e2e-gate.mjs',
  'tests/gates/p9/task-batch-7-e2e.mjs',
  'package.json',
];

function rootPath(rel) { return path.join(REPO_ROOT, rel); }
function read(rel) { try { return fs.readFileSync(rootPath(rel), 'utf8'); } catch { return ''; } }
function readJSON(rel) { try { return JSON.parse(read(rel)); } catch { return null; } }
function write(rel, value) {
  fs.mkdirSync(path.dirname(rootPath(rel)), { recursive: true });
  const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  fs.writeFileSync(rootPath(rel), content.endsWith('\n') ? content : `${content}\n`, 'utf8');
}
function sha256Buffer(value) { return crypto.createHash('sha256').update(value).digest('hex'); }
function sha256File(rel) { return fs.existsSync(rootPath(rel)) ? sha256Buffer(fs.readFileSync(rootPath(rel))) : ''; }
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}
function hasAll(text, values) { return values.every((value) => String(text || '').includes(value)); }
function secretLeakFree(values) {
  return values.every((value) => !/(authorization|cookie|access_token|refresh_token|client_secret|idempotency-key\s*[:=]\s*[a-z0-9_-]{12,}|postgres(?:ql)?:\/\/[^\s"']+)/i.test(String(value || '')));
}

export function p9Batch7SourceManifest() {
  const currentBranch = git(['branch', '--show-current']);
  const currentHead = git(['rev-parse', 'HEAD']);
  const files = REQUIRED_FILES
    .filter((file) => !file.startsWith('artifacts/') && !file.endsWith('-gate.json') && !file.endsWith('_GATE.md'))
    .filter((file) => file !== P9_BATCH_7_JSON && file !== P9_BATCH_7_MD)
    .filter((file) => fs.existsSync(rootPath(file)))
    .sort();
  const hash = crypto.createHash('sha256');
  const entries = files.map((file) => {
    const sha256 = sha256File(file);
    hash.update(file);
    hash.update('\0');
    hash.update(sha256);
    hash.update('\n');
    return { path: file, sha256 };
  });
  return { schemaVersion: 1, phase: 'P9', batchId: 'P9-TASK-BATCH-7', generatedAt: new Date().toISOString(), currentBranch, currentHead, sha256: hash.digest('hex'), fileCount: entries.length, entries };
}

export function validateP9Batch7IntegrationBundle({ evidence = {}, runtime = {}, postgresRuntime = {}, postgresGate = {}, batch6Gate = {}, sources = {}, gitState = {}, requiredFilesPresent, sourceManifest: injectedSourceManifest, liveSourceManifest: injectedLiveSourceManifest, protectedSourceFreeze: injectedProtectedSourceFreeze, liveProtectedSourceManifest: injectedLiveProtectedSourceManifest, runtimeArtifactSha: injectedRuntimeArtifactSha, e2eArtifactSha: injectedE2EArtifactSha, runtimeJsonlSha: injectedRuntimeJsonlSha, raceArtifactSha: injectedRaceArtifactSha } = {}) {
  const branch = gitState.currentBranch ?? git(['branch', '--show-current']);
  const head = gitState.currentHead ?? git(['rev-parse', 'HEAD']);
  const detached = gitState.headDetached ?? git(['rev-parse', '--abbrev-ref', 'HEAD']) === 'HEAD';
  const staged = gitState.stagedFileCount ?? (git(['diff', '--cached', '--name-only']) ? git(['diff', '--cached', '--name-only']).split(/\r?\n/).filter(Boolean).length : 0);
  const e2e = sources.e2e ?? read('admin/e2e/specs/p9-inventory-sync.spec.ts');
  const mocks = sources.mocks ?? read('admin/e2e/mocks/inventory-sync-p9.ts');
  const service = sources.service ?? read('admin/src/services/inventorySyncP9.ts');
  const backendIntegration = sources.backendIntegration ?? [
    read('backend/internal/testing/integration/p9_postgres_integration_test.go'),
    read('backend/internal/modules/inventorysyncp9/postgres_integration_test.go'),
    read('backend/internal/modules/inventorysyncp9/postgres_contract_test.go'),
  ].join('\n');
  const packageJSON = sources.packageJSON ?? read('package.json');
  const sourceManifest = injectedSourceManifest || readJSON(P9_BATCH_7_SOURCE_MANIFEST_JSON) || {};
  const liveSourceManifest = injectedLiveSourceManifest || p9Batch7SourceManifest();
  const protectedSourceFreeze = injectedProtectedSourceFreeze || readProtectedSourceFreeze();
  const liveProtectedSourceManifest = injectedLiveProtectedSourceManifest || computeLiveProtectedSourceManifest();
  const protectedSourceValidation = validateProtectedSourceFreezeBundle({
    freeze: protectedSourceFreeze,
    live: liveProtectedSourceManifest,
    gitState: { currentBranch: branch, currentHead: head },
  });
  const runtimeArtifactSha = injectedRuntimeArtifactSha ?? sha256File(P9_BATCH_7_RUNTIME_JSON);
  const e2eArtifactSha = injectedE2EArtifactSha ?? sha256File(P9_BATCH_7_E2E_JSONL);
  const runtimeJsonlSha = injectedRuntimeJsonlSha ?? sha256File(P9_BATCH_7_RUNTIME_JSONL);
  const raceArtifactSha = injectedRaceArtifactSha ?? sha256File(P9_BATCH_7_RACE_JSONL);
  const checks = [
    ['requiredFilesPresent', requiredFilesPresent ?? REQUIRED_FILES.every((file) => fs.existsSync(rootPath(file)))],
    ['packageScriptsPresent', hasAll(packageJSON, ['test:p9-task-batch-7-e2e', 'test:p9-task-batch-7-runtime', 'p9:task-batch-7-e2e-gate'])],
    ['currentBranch', branch === 'dev'],
    ['headDetached', detached === false],
    ['stagedFileCount', staged === 0],
    ['batchId', evidence.batchId === 'P9-TASK-BATCH-7' && runtime.batchId === 'P9-TASK-BATCH-7'],
    ['status', evidence.status === 'completed' && evidence.integrationStatus === 'passed' && runtime.status === 'passed' && runtime.completed === true],
    ['runtimeHeadBinding', evidence.currentBranch === branch && evidence.currentHead === head && runtime.currentBranch === branch && runtime.currentHead === head],
    ...TASK_IDS.map((id) => [`${id} status`, evidence.tasks?.[id]?.status === 'completed']),
    ['formalTaskCount', evidence.formalTaskTotal === 5 && evidence.formalTaskCompletedCount === 5],
    ['acceptanceCoverage', ACCEPTANCE_IDS.every((id) => evidence.acceptanceCriteriaPassedIds?.includes(id))],
    ['runtimeBinding', evidence.runtimeEvidence?.runId === runtime.runId && evidence.runtimeEvidence?.finishedAt === runtime.finishedAt && evidence.runtimeEvidence?.runtimeSummarySha256 === runtimeArtifactSha],
    ['sourceManifestBinding', evidence.runtimeEvidence?.sourceManifestSha256 === sourceManifest.sha256 && runtime.sourceManifestSha256 === sourceManifest.sha256 && sourceManifest.currentBranch === branch && sourceManifest.currentHead === head && liveSourceManifest.sha256 === sourceManifest.sha256],
    ['protectedSourceManifestBinding', protectedSourceValidation.status === 'passed'
      && evidence.runtimeEvidence?.protectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && evidence.postgresRuntimeEvidence?.protectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && runtime.protectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && runtime.protectedSourceFreezeHead === head
      && runtime.protectedSourceManifestBeforeSha256 === protectedSourceFreeze.sha256
      && runtime.protectedSourceManifestAfterSha256 === protectedSourceFreeze.sha256
      && runtime.protectedSourceFrozen === true
      && runtime.protectedSourceStable === true
      && runtime.protectedSourceDriftDetected === false
      && postgresRuntime.protectedSourceFreeze?.sha256 === protectedSourceFreeze.sha256],
    ['artifactHashBinding', evidence.runtimeEvidence?.e2eArtifactSha256 === e2eArtifactSha && runtime.e2eArtifactSha256 === e2eArtifactSha && evidence.runtimeEvidence?.runtimeJsonlSha256 === runtimeJsonlSha && runtime.runtimeJsonlSha256 === runtimeJsonlSha && evidence.runtimeEvidence?.raceArtifactSha256 === raceArtifactSha && runtime.raceArtifactSha256 === raceArtifactSha],
    ['postgresRuntimeBinding', evidence.postgresRuntimeEvidence?.runId === postgresRuntime.runId && postgresRuntime.git?.endHead === head && postgresGate.status === 'passed' && postgresGate.currentHead === head],
    ['postgresContracts', postgresRuntime.contracts?.postgresIntegrationPassed === true && postgresRuntime.contracts?.postgresFixtureGoldenPathPassed === true],
    ['authenticatedE2E', runtime.authenticatedPostgresE2E === true && backendIntegration.includes('TestP9PGBearerAuthAndFixtureGoldenPath')],
    ['tenantRbacIsolation', runtime.tenantIsolationPassed === true && runtime.rbacMatrixPassed === true && runtime.crossTenantWriteDenied === true],
    ['idempotencyAndConflict', runtime.idempotencyPassed === true && runtime.revisionConflictPassed === true],
    ['keysetPagination', runtime.keysetPaginationPassed === true && service.includes('nextCursor') && !/\boffset\b/i.test(service)],
    ['adminE2E', runtime.adminE2EPassed === true && batch6Gate.status === 'passed' && e2e.includes('@p9-batch7') && e2e.includes('viewports') && e2e.includes('writeGuard.allow')],
    ['integrationFixtures', runtime.integrationFixtures?.success === true && runtime.integrationFixtures?.lowConfidence === true && runtime.integrationFixtures?.conflict === true && runtime.integrationFixtures?.manualBinding === true && runtime.integrationFixtures?.failure === true && mocks.includes('success_single_page')],
    ['platformBoundaryFinalGate', runtime.platformBoundary?.realPlatformNetworkCalls === 0 && runtime.platformBoundary?.realCredentialsUsed === false && runtime.platformBoundary?.inventoryMutationCalls === 0],
    ['noProductCapabilityAdded', Array.isArray(evidence.integrationDefectFixes) && evidence.integrationDefectFixes.length === 0],
    ['p10BoundaryPreserved', evidence.p10BoundaryPreserved === true && runtime.p10BoundaryPreserved === true],
    ['productionNotReady', evidence.productionReady === false && evidence.productionAcceptancePassed === false && runtime.productionReady === false && runtime.productionAcceptancePassed === false],
    ['noRealPlatformCapability', evidence.realDouyinProviderImplemented === false && evidence.oauthImplemented === false && evidence.realPlatformReadEnabled === false && evidence.realPlatformWriteEnabled === false && evidence.workerImplemented === false && evidence.backgroundSyncWorkerImplemented === false && evidence.automaticRetryWorkerImplemented === false],
    ['secretLeakFree', secretLeakFree([JSON.stringify(evidence), JSON.stringify(runtime), read(P9_BATCH_7_MD), read(P9_BATCH_7_RUNTIME_JSONL), read(P9_BATCH_7_E2E_JSONL), read(P9_BATCH_7_RACE_JSONL), JSON.stringify(sourceManifest)])],
  ];
  const failed = checks.filter(([, ok]) => !ok).map(([id]) => id);
  return {
    phase: 'P9',
    gate: 'P9-TASK-BATCH-7-E2E',
    status: failed.length === 0 ? 'passed' : 'failed',
    checkedAt: new Date().toISOString(),
    failed,
    failedCount: failed.length,
    currentBranch: branch,
    currentHead: head,
    headDetached: detached,
    stagedFileCount: staged,
    runtimeRunId: runtime.runId || '',
    runtimeSummarySha256: runtimeArtifactSha,
    sourceManifestSha256: sourceManifest.sha256 || '',
    protectedSourceManifestSha256: protectedSourceFreeze.sha256 || '',
    protectedSourceDriftDetected: protectedSourceValidation.protectedSourceDriftDetected,
    postgresRuntimeRunId: postgresRuntime.runId || '',
    productionReady: evidence.productionReady === true,
    productionAcceptancePassed: evidence.productionAcceptancePassed === true,
    p10BoundaryPreserved: evidence.p10BoundaryPreserved === true,
    integrationDefectFixes: evidence.integrationDefectFixes || [],
    checks: checks.map(([id, ok]) => ({ id, status: ok ? 'passed' : 'failed' })),
  };
}

export function renderP9Batch7GateMarkdown(report) {
  return `# P9 Task Batch 7 Integration Development Closure Gate

Status: **${report.status}**

- Runtime run ID: ${report.runtimeRunId || 'missing'}
- Runtime summary SHA-256: ${report.runtimeSummarySha256 || 'missing'}
- Source manifest SHA-256: ${report.sourceManifestSha256 || 'missing'}
- Protected source manifest SHA-256: ${report.protectedSourceManifestSha256 || 'missing'}
- Protected source drift detected: ${report.protectedSourceDriftDetected}
- PostgreSQL runtime run ID: ${report.postgresRuntimeRunId || 'missing'}
- Current branch: ${report.currentBranch}
- Current HEAD: ${report.currentHead}
- Staged files: ${report.stagedFileCount}
- Production ready: ${report.productionReady}
- Production acceptance passed: ${report.productionAcceptancePassed}
- P10 boundary preserved: ${report.p10BoundaryPreserved}
- Integration defect fixes: ${report.integrationDefectFixes.length}
- Failed checks: ${report.failedCount ? report.failed.join(', ') : 'none'}

${report.checks.map((check) => `- ${check.status === 'passed' ? 'PASS' : 'FAIL'} \`${check.id}\``).join('\n')}
`;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  const report = validateP9Batch7IntegrationBundle({
    evidence: readJSON(P9_BATCH_7_JSON) || {},
    runtime: readJSON(P9_BATCH_7_RUNTIME_JSON) || {},
    postgresRuntime: readJSON('artifacts/p9-postgres-runtime.json') || {},
    postgresGate: readJSON('docs/p9-postgresql-integration-closure-gate.json') || {},
    batch6Gate: readJSON('docs/p9-task-batch-6-admin-inventory-center-gate.json') || {},
  });
  write(P9_BATCH_7_GATE_JSON, report);
  write(P9_BATCH_7_GATE_MD, renderP9Batch7GateMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.status !== 'passed') process.exitCode = 1;
}
