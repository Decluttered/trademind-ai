import fs from 'node:fs';
import crypto from 'node:crypto';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  P9_BATCH_7_GATE_JSON,
  P9_BATCH_7_JSON,
  P9_BATCH_7_RUNTIME_JSON,
  validateP9Batch7IntegrationBundle,
} from './p9-task-batch-7-e2e-gate.mjs';
import {
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  computeLiveProtectedSourceManifest,
  readProtectedSourceFreeze,
  validateProtectedSourceFreezeBundle,
} from './p9-protected-source-freeze.mjs';

export const P9_FINAL_CLOSURE_JSON = 'docs/p9-final-development-closure.json';
export const P9_FINAL_CLOSURE_MD = 'docs/P9_FINAL_DEVELOPMENT_CLOSURE.md';
export const P9_FINAL_CLOSURE_GATE_JSON = 'docs/p9-final-development-closure-gate.json';
export const P9_FINAL_CLOSURE_GATE_MD = 'docs/P9_FINAL_DEVELOPMENT_CLOSURE_GATE.md';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const PRODUCT_TASK_IDS = [
  'P9-501', 'P9-502', 'P9-503', 'P9-504', 'P9-505', 'P9-506',
  'P9-601', 'P9-602', 'P9-603', 'P9-604', 'P9-605', 'P9-606',
  'P9-701', 'P9-702', 'P9-703', 'P9-704', 'P9-705', 'P9-706',
  'P9-801', 'P9-802', 'P9-803', 'P9-804',
  'P9-901', 'P9-902', 'P9-903', 'P9-904', 'P9-905',
  'P9-1001', 'P9-1002', 'P9-1003', 'P9-1004', 'P9-1005', 'P9-1006',
  'P9-1101', 'P9-1102', 'P9-1103', 'P9-1104', 'P9-1105',
];
const ACCEPTANCE_IDS = Array.from({ length: 15 }, (_, i) => `AC-P9-${String(i + 1).padStart(2, '0')}`);
const HISTORICAL_GATE_PATHS = [
  'docs/p9-entry-gate-report.json',
  'docs/p9-plan-final-gate.json',
  'docs/p9-task-batch-1-scope-gate.json',
  'docs/p9-task-batch-1-domain-persistence-gate.json',
  'docs/p9-task-batch-2-sku-calibration-gate.json',
  'docs/p9-task-batch-3-sync-orchestration-gate.json',
  'docs/p9-task-batch-4-permissions-audit-safety-gate.json',
  'docs/p9-task-batch-5-backend-apis-gate.json',
  'docs/p9-postgresql-integration-closure-gate.json',
  'docs/p9-task-batch-6-admin-inventory-center-gate.json',
  P9_BATCH_7_GATE_JSON,
];
const REQUIRED_FILES = [
  P9_FINAL_CLOSURE_JSON,
  P9_FINAL_CLOSURE_MD,
  P9_BATCH_7_JSON,
  P9_BATCH_7_GATE_JSON,
  P9_BATCH_7_RUNTIME_JSON,
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  'docs/P9_CURRENT_HEAD_RECLOSURE.md',
  'docs/p9-current-head-reclosure.json',
  'docs/P9_EXECUTION_PLAN.md',
  'docs/p9-execution-plan.json',
  'docs/PROGRESS.md',
  'docs/README.md',
  'docs/P9_OWNER_SCOPE_DECISION.md',
  'docs/p9-owner-scope-decision.json',
];

function rootPath(rel) { return path.join(REPO_ROOT, rel); }
function read(rel) { try { return fs.readFileSync(rootPath(rel), 'utf8'); } catch { return ''; } }
function readJSON(rel) { try { return JSON.parse(read(rel)); } catch { return null; } }
function write(rel, value) {
  fs.mkdirSync(path.dirname(rootPath(rel)), { recursive: true });
  const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  fs.writeFileSync(rootPath(rel), content.endsWith('\n') ? content : `${content}\n`, 'utf8');
}
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}
function flattenTasks(plan = {}) {
  return (plan.workstreams || []).flatMap((ws) => (ws.tasks || []).map((task) => ({ ...task, workstreamId: ws.workstreamId, batch: task.batch ?? ws.batch })));
}
function unique(values) { return [...new Set(values.filter(Boolean))]; }
function sameSet(actual, expected) {
  const a = [...actual].map(String).sort();
  const b = [...expected].map(String).sort();
  return a.length === b.length && a.every((value, index) => value === b[index]);
}
function secretLeakFree(values) {
  return values.every((value) => !/(authorization|cookie|access_token|refresh_token|client_secret|idempotency-key\s*[:=]\s*[a-z0-9_-]{12,}|postgres(?:ql)?:\/\/[^\s"']+)/i.test(String(value || '')));
}
function gateStatusPassed(gatePath, report) {
  return report.status === 'passed' || (gatePath === 'docs/p9-entry-gate-report.json' && report.status === 'allowed');
}
function sha256File(rel) {
  return fs.existsSync(rootPath(rel)) ? crypto.createHash('sha256').update(fs.readFileSync(rootPath(rel))).digest('hex') : '';
}

export function validateP9FinalDevelopmentClosureBundle({ closure = {}, plan = {}, batch7Evidence = {}, batch7Runtime = {}, postgresRuntime: injectedPostgresRuntime, gateReports = {}, gitState = {}, requiredFilesPresent, protectedSourceFreeze: injectedProtectedSourceFreeze, liveProtectedSourceManifest: injectedLiveProtectedSourceManifest, artifactHashes = {} } = {}) {
  const branch = gitState.currentBranch ?? git(['branch', '--show-current']);
  const head = gitState.currentHead ?? git(['rev-parse', 'HEAD']);
  const detached = gitState.headDetached ?? git(['rev-parse', '--abbrev-ref', 'HEAD']) === 'HEAD';
  const staged = gitState.stagedFileCount ?? (git(['diff', '--cached', '--name-only']) ? git(['diff', '--cached', '--name-only']).split(/\r?\n/).filter(Boolean).length : 0);
  const tasks = flattenTasks(plan);
  const productTasks = tasks.filter((task) => task.taskCategory === 'product_implementation' || task.planningFoundation === false);
  const productTaskIds = productTasks.map((task) => task.taskId);
  const productCompletedIds = productTasks.filter((task) => task.status === 'completed').map((task) => task.taskId);
  const planAcceptanceIds = unique(productTasks.flatMap((task) => task.acceptanceCriteriaIds || []));
  const historicalGateRows = HISTORICAL_GATE_PATHS.map((gatePath) => ({ path: gatePath, report: gateReports[gatePath] || readJSON(gatePath) || {} }));
  const postgresRuntime = injectedPostgresRuntime || readJSON('artifacts/p9-postgres-runtime.json') || {};
  const protectedSourceFreeze = injectedProtectedSourceFreeze || readProtectedSourceFreeze();
  const liveProtectedSourceManifest = injectedLiveProtectedSourceManifest || computeLiveProtectedSourceManifest();
  const protectedSourceValidation = validateProtectedSourceFreezeBundle({
    freeze: protectedSourceFreeze,
    live: liveProtectedSourceManifest,
    gitState: { currentBranch: branch, currentHead: head },
  });
  const postgresRuntimeSha256 = artifactHashes.postgresRuntimeSha256 || sha256File('artifacts/p9-postgres-runtime.json');
  const batch7RuntimeSha256 = artifactHashes.batch7RuntimeSha256 || sha256File(P9_BATCH_7_RUNTIME_JSON);
  const batch7Validation = gateReports.batch7Validation || validateP9Batch7IntegrationBundle({
    evidence: batch7Evidence,
    runtime: batch7Runtime,
    postgresRuntime: readJSON('artifacts/p9-postgres-runtime.json') || {},
    postgresGate: readJSON('docs/p9-postgresql-integration-closure-gate.json') || {},
    batch6Gate: readJSON('docs/p9-task-batch-6-admin-inventory-center-gate.json') || {},
    gitState: { currentBranch: branch, currentHead: head, headDetached: detached, stagedFileCount: staged },
  });
  const finalFlags = [
    closure.productionReady,
    closure.productionAcceptancePassed,
    closure.realDouyinProviderImplemented,
    closure.oauthImplemented,
    closure.realPlatformReadEnabled,
    closure.realPlatformWriteEnabled,
    closure.inventoryMutationEnabled,
    closure.workerImplemented,
    closure.backgroundSyncWorkerImplemented,
    closure.automaticRetryWorkerImplemented,
    closure.productionTagCreated,
    closure.releaseCreated,
    plan.productionReady,
    plan.productionAcceptancePassed,
  ];
  const checks = [
    ['requiredFilesPresent', requiredFilesPresent ?? REQUIRED_FILES.every((file) => fs.existsSync(rootPath(file)))],
    ['currentBranch', branch === 'dev'],
    ['headDetached', detached === false],
    ['stagedFileCount', staged === 0],
    ['closureStatus', closure.status === 'passed' && closure.developmentClosureStatus === 'passed' && closure.p9Complete === true && closure.developmentClosurePassed === true],
    ['closureHeadBinding', closure.currentBranch === branch && closure.currentHead === head && closure.currentClosureHead === head && closure.currentHeadClosureVerified === true],
    ['previousClosurePreserved', Array.isArray(closure.previousClosures)
      ? closure.previousClosures.some((item) => item.head === closure.previousClosureHead && item.head !== head && item.status === 'passed')
      : closure.previousClosureHead === closure.initialClosure?.head && closure.previousClosureHead !== head && closure.initialClosure?.status === 'passed'],
    ['planStatus', plan.phaseStatus === 'Development Complete' && plan.executionStatus === 'development_complete' && plan.p9Complete === true && plan.p9DevelopmentClosurePassed === true],
    ['productTaskIdsPreserved', sameSet(productTaskIds, PRODUCT_TASK_IDS)],
    ['productTasksCompleted', sameSet(productCompletedIds, PRODUCT_TASK_IDS) && plan.productCompletedTaskCount === 38],
    ['batch7Completed', plan.batch7Completed === true && batch7Evidence.status === 'completed' && batch7Validation.status === 'passed'],
    ['runtimeHeadBindings', closure.postgresRuntimeRunId === postgresRuntime.runId && closure.postgresRuntimeHead === head && postgresRuntime.git?.endHead === head && closure.batch7RuntimeRunId === batch7Runtime.runId && closure.batch7RuntimeHead === head && batch7Runtime.currentHead === head && batch7Evidence.currentHead === head],
    ['runtimeArtifactBindings', closure.postgresRuntimeSummarySha256 === postgresRuntimeSha256 && closure.batch7RuntimeSummarySha256 === batch7RuntimeSha256],
    ['protectedSourceManifestBindings', protectedSourceValidation.status === 'passed'
      && closure.protectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && closure.postgresRuntimeProtectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && closure.batch7RuntimeProtectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && postgresRuntime.protectedSourceFreeze?.sha256 === protectedSourceFreeze.sha256
      && batch7Runtime.protectedSourceManifestSha256 === protectedSourceFreeze.sha256
      && closure.protectedSourceDriftDetected === false
      && closure.currentReclosure?.protectedSourceManifestSha256 === protectedSourceFreeze.sha256],
    ['acceptanceCriteriaPassed', sameSet(planAcceptanceIds, ACCEPTANCE_IDS) && sameSet(closure.acceptanceCriteriaPassedIds || [], ACCEPTANCE_IDS) && closure.acceptanceCriteriaPassedCount === 15],
    ['historicalGatesPassed', historicalGateRows.every(({ path: gatePath, report }) => gateStatusPassed(gatePath, report))],
    ['postgresIntegrationPassed', plan.postgresIntegrationPassed === true && closure.postgresIntegrationPassed === true],
    ['adminE2EPassed', closure.adminE2EPassed === true && closure.adminResponsiveE2EPassed === true],
    ['qualityGateEvidenceRecorded', Array.isArray(closure.validationCommands) && closure.validationCommands.length > 0],
    ['p10BoundaryPreserved', closure.p10BoundaryPreserved === true && plan.p10BoundaryPreserved === true],
    ['productionBoundary', finalFlags.every((value) => value === false || value === undefined)],
    ['integrationDefectFixes', Array.isArray(closure.integrationDefectFixes) && closure.integrationDefectFixes.length === 0],
    ['secretLeakFree', secretLeakFree([JSON.stringify(closure), JSON.stringify(plan), read(P9_FINAL_CLOSURE_MD), read(P9_BATCH_7_RUNTIME_JSON)])],
  ];
  const failed = checks.filter(([, ok]) => !ok).map(([id]) => id);
  return {
    phase: 'P9',
    gate: 'P9-FINAL-DEVELOPMENT-CLOSURE',
    status: failed.length === 0 ? 'passed' : 'failed',
    checkedAt: new Date().toISOString(),
    failed,
    failedCount: failed.length,
    currentBranch: branch,
    currentHead: head,
    headDetached: detached,
    stagedFileCount: staged,
    productTaskTotal: productTasks.length,
    productCompletedTaskCount: productCompletedIds.length,
    acceptanceCriteriaTotal: planAcceptanceIds.length,
    acceptanceCriteriaPassedCount: closure.acceptanceCriteriaPassedCount || 0,
    historicalGateFailureCount: historicalGateRows.filter(({ path: gatePath, report }) => !gateStatusPassed(gatePath, report)).length,
    currentHeadClosureVerified: closure.currentHeadClosureVerified === true,
    closureHeadBinding: closure.currentClosureHead === head,
    runtimeHeadBindings: closure.postgresRuntimeHead === head && closure.batch7RuntimeHead === head,
    protectedSourceManifestBindings: closure.protectedSourceManifestSha256 === protectedSourceFreeze.sha256,
    protectedSourceManifestSha256: protectedSourceFreeze.sha256 || '',
    currentProtectedSourceManifestSha256: liveProtectedSourceManifest.sha256 || '',
    protectedSourceDriftDetected: protectedSourceValidation.protectedSourceDriftDetected || closure.protectedSourceDriftDetected === true,
    p9Complete: closure.p9Complete === true,
    developmentClosurePassed: closure.developmentClosurePassed === true,
    productionReady: closure.productionReady === true,
    productionAcceptancePassed: closure.productionAcceptancePassed === true,
    p10BoundaryPreserved: closure.p10BoundaryPreserved === true,
    checks: checks.map(([id, ok]) => ({ id, status: ok ? 'passed' : 'failed' })),
  };
}

export function renderP9FinalDevelopmentClosureGateMarkdown(report) {
  return `# P9 Final Development Closure Gate

Status: **${report.status}**

- Current branch: ${report.currentBranch}
- Current HEAD: ${report.currentHead}
- Staged files: ${report.stagedFileCount}
- Product tasks: ${report.productCompletedTaskCount}/${report.productTaskTotal}
- Acceptance criteria: ${report.acceptanceCriteriaPassedCount}/${report.acceptanceCriteriaTotal}
- Historical gate failures: ${report.historicalGateFailureCount}
- Protected source manifest SHA-256: ${report.protectedSourceManifestSha256 || 'missing'}
- Protected source drift detected: ${report.protectedSourceDriftDetected}
- P9 complete: ${report.p9Complete}
- Development closure passed: ${report.developmentClosurePassed}
- Production ready: ${report.productionReady}
- Production acceptance passed: ${report.productionAcceptancePassed}
- P10 boundary preserved: ${report.p10BoundaryPreserved}
- Failed checks: ${report.failedCount ? report.failed.join(', ') : 'none'}

${report.checks.map((check) => `- ${check.status === 'passed' ? 'PASS' : 'FAIL'} \`${check.id}\``).join('\n')}
`;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  const report = validateP9FinalDevelopmentClosureBundle({
    closure: readJSON(P9_FINAL_CLOSURE_JSON) || {},
    plan: readJSON('docs/p9-execution-plan.json') || {},
    batch7Evidence: readJSON(P9_BATCH_7_JSON) || {},
    batch7Runtime: readJSON(P9_BATCH_7_RUNTIME_JSON) || {},
    postgresRuntime: readJSON('artifacts/p9-postgres-runtime.json') || {},
  });
  write(P9_FINAL_CLOSURE_GATE_JSON, report);
  write(P9_FINAL_CLOSURE_GATE_MD, renderP9FinalDevelopmentClosureGateMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.status !== 'passed') process.exitCode = 1;
}
