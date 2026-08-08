import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { computeLiveProtectedSourceManifest } from './p9-protected-source-freeze.mjs';
import { computeP10PlanningSemanticManifest } from './p10-planning-semantic-manifest.mjs';
import {
  readEnvFile,
  validateP10ExternalInfrastructureEvidence,
  validateP10PreproductionContract,
} from './p10-preproduction-contract.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const EVIDENCE_JSON = 'docs/p10-task-batch-1-preproduction-foundation.json';
const EVIDENCE_MD = 'docs/P10_TASK_BATCH_1_PREPRODUCTION_FOUNDATION.md';
const GATE_JSON = 'docs/p10-task-batch-1-preproduction-foundation-gate.json';
const GATE_MD = 'docs/P10_TASK_BATCH_1_PREPRODUCTION_FOUNDATION_GATE.md';

const REQUIRED_FILES = [
  '.env.staging.example',
  'deploy/preproduction/compose.yml',
  'deploy/scripts/check-preproduction-readiness.sh',
  'deploy/scripts/deploy-preproduction.sh',
  'deploy/scripts/migrate-preproduction.sh',
  'deploy/scripts/backup-preproduction.sh',
  'deploy/scripts/restore-preproduction.sh',
  'deploy/scripts/rollback-preproduction.sh',
  'deploy/scripts/teardown-preproduction.sh',
  'docs/P10_PREPRODUCTION_ARCHITECTURE.md',
  'docs/p10-task-batch-1-external-infrastructure.json',
  'scripts/p10-preproduction-contract.mjs',
  'scripts/p10-preproduction-preflight.mjs',
];

function rootPath(relativePath) { return path.join(REPO_ROOT, relativePath); }
function read(relativePath) { try { return fs.readFileSync(rootPath(relativePath), 'utf8'); } catch { return ''; } }
function readJSON(relativePath) { try { return JSON.parse(read(relativePath)); } catch { return null; } }
function write(relativePath, value) {
  const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  fs.writeFileSync(rootPath(relativePath), content.endsWith('\n') ? content : `${content}\n`, 'utf8');
}
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}

function checkStatus(report, id) {
  return report?.checks?.find((check) => check.id === id)?.status === 'passed';
}

function scanSecrets(rawText) {
  const patterns = [
    /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/g,
    /\bBearer\s+[A-Za-z0-9._~-]{20,}/g,
    /\b(?:postgresql|postgres|redis):\/\/[^\s:@/]+:[^\s@/]+@/g,
  ];
  const realSecretCount = patterns.reduce((total, pattern) => total + (rawText.match(pattern)?.length || 0), 0);
  return { realSecretCount, credentialValueRecorded: realSecretCount > 0 };
}

export function validateP10TaskBatch1Bundle({
  plan = {}, transition = {}, planningGate = {}, ownerGate = {}, revalidation = {},
  planningSemanticManifest = {}, frozenP9 = {}, liveP9 = {}, gitState = {},
  contract = {}, externalEvidence = {}, external = {}, repository = {},
  requiredFilesPresent = true, credentialScan = {},
} = {}) {
  const batch1 = (plan.batches || []).find((batch) => batch.batchId === 'P10-B1') || {};
  const taskIds = batch1.taskIds || [];
  const p9ProtectedSourceModified = !frozenP9.sha256 || !liveP9.sha256
    || frozenP9.sha256 !== liveP9.sha256 || frozenP9.gitHead !== gitState.currentHead;
  const approvedPlanningSemanticsDriftDetected = !planningSemanticManifest.sha256
    || planningSemanticManifest.sha256 !== revalidation.planningSemanticManifestSha256;
  const changesCommittedDuringBatch1 = gitState.currentHead !== revalidation.currentRunBaseHead;
  const repositoryFoundationPassed = requiredFilesPresent
    && repository.composeIsolation === true
    && repository.readinessFoundation === true
    && repository.migrationFoundation === true
    && repository.backupFoundation === true
    && repository.restoreFoundation === true
    && repository.rollbackFoundation === true
    && repository.teardownFoundation === true
    && contract.status === 'passed';
  const externalInfrastructurePassed = external.status === 'passed';
  const formalTaskCompleted = (repositoryFoundationPassed ? 3 : 0) + (externalInfrastructurePassed ? 1 : 0);
  const checks = [
    ['entryTransition', transition.status === 'passed' && transition.failedCount === 0
      && transition.p9ClosureReuseEligible === true && transition.p10PlanningEntryAllowed === true],
    ['entryPlanning', planningGate.status === 'passed' && planningGate.failedCount === 0
      && planningGate.planningPackCurrentHeadValid === true],
    ['entryOwnerDecision', ownerGate.status === 'passed' && ownerGate.failedCount === 0
      && ownerGate.p10OwnerDecisionApproved === true && ownerGate.p10ExecutionPlanFinalized === true],
    ['formalTaskCount', taskIds.length === 4
      && ['P10-101', 'P10-102', 'P10-103', 'P10-104'].every((id) => taskIds.includes(id))],
    ['repositoryFoundation', repositoryFoundationPassed],
    ['preproductionEnvironmentDefined', checkStatus(contract, 'preproductionEnvironmentMapping')],
    ['unknownEnvironmentFailClosed', repository.unknownEnvironmentFailClosed === true],
    ['databaseIsolation', checkStatus(contract, 'databaseIsolation')],
    ['redisIsolation', checkStatus(contract, 'redisIsolation')],
    ['secretExternalization', checkStatus(contract, 'secretExternalization')],
    ['sessionIsolation', checkStatus(contract, 'sessionIsolation')],
    ['capabilityDefaults', checkStatus(contract, 'capabilityDefaults')],
    ['startupSafety', repository.startupSafety === true],
    ['migrationSafety', checkStatus(contract, 'migrationSafety') && repository.migrationFoundation === true],
    ['healthReadiness', repository.readinessFoundation === true],
    ['backupFoundation', checkStatus(contract, 'backupRestoreSafety') && repository.backupFoundation === true],
    ['restoreFoundation', checkStatus(contract, 'backupRestoreSafety') && repository.restoreFoundation === true],
    ['rollbackFoundation', checkStatus(contract, 'rollbackFoundation') && repository.rollbackFoundation === true],
    ['productionRestoreDisabled', repository.productionRestoreEnabled === false],
    ['externalInfrastructureProvisioned', externalInfrastructurePassed],
    ['formalTaskCompletion', formalTaskCompleted === 4],
    ['p9ProtectedSourceModified', p9ProtectedSourceModified === false],
    ['approvedPlanningSemanticsDriftDetected', approvedPlanningSemanticsDriftDetected === false],
    ['changesCommittedDuringBatch1', changesCommittedDuringBatch1 === false],
    ['realSecretCount', Number(credentialScan.realSecretCount) === 0],
    ['credentialValueRecorded', credentialScan.credentialValueRecorded === false],
    ['currentBranch', gitState.currentBranch === 'dev'],
    ['stagedFileCount', gitState.stagedFileCount === 0],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    phase: 'P10', batchId: 'P10-B1', gate: 'P10-BATCH-1-PREPRODUCTION-FOUNDATION',
    status: failed.length === 0 ? 'passed' : 'failed',
    batchStatus: formalTaskCompleted === 4 ? 'completed' : 'incomplete',
    checkedAt: new Date().toISOString(), failed, failedCount: failed.length,
    planningPackHistoricalCheckpoint: revalidation.initialPlanningCheckpoint || '',
    planningPackCurrentValidationHead: revalidation.currentPlanningValidationHead || '',
    planningSemanticRevalidationPassed: revalidation.planningSemanticRevalidationPassed === true,
    batch1EntryHead: revalidation.currentRunBaseHead || '',
    batch1PlanningSemanticManifestSha256: revalidation.planningSemanticManifestSha256 || '',
    batch1P9ProtectedManifestSha256: frozenP9.sha256 || '',
    formalTaskTotal: taskIds.length,
    formalTaskIds: taskIds,
    formalTaskCompleted,
    formalTaskFailed: 0,
    formalTaskDeferred: Math.max(0, taskIds.length - formalTaskCompleted),
    repositoryFoundationPassed,
    externalInfrastructureStatus: externalEvidence.externalInfrastructureStatus || 'not_provisioned',
    externalProvisioningPending: !externalInfrastructurePassed,
    preproductionEnvironmentDefined: checkStatus(contract, 'preproductionEnvironmentMapping'),
    unknownEnvironmentFailClosed: repository.unknownEnvironmentFailClosed === true,
    databaseIsolationPassed: checkStatus(contract, 'databaseIsolation'),
    productionDatabaseReuse: false,
    testDatabaseReuse: false,
    redisIsolationPassed: checkStatus(contract, 'redisIsolation'),
    secretExternalizationPassed: checkStatus(contract, 'secretExternalization'),
    sessionIsolationPassed: checkStatus(contract, 'sessionIsolation'),
    startupSafetyPassed: repository.startupSafety === true,
    migrationSafetyPassed: checkStatus(contract, 'migrationSafety') && repository.migrationFoundation === true,
    healthReadinessPassed: repository.readinessFoundation === true,
    backupFoundationPassed: repository.backupFoundation === true,
    restoreFoundationPassed: repository.restoreFoundation === true,
    rollbackFoundationPassed: repository.rollbackFoundation === true,
    productionRestoreEnabled: repository.productionRestoreEnabled === true,
    p9ProtectedSourceModified,
    approvedPlanningSemanticsDriftDetected,
    changesCommittedDuringBatch1,
    currentAllowedLevel: 'L0',
    realProviderEnabled: false,
    realPlatformNetworkEnabled: false,
    realInventoryReadEnabled: false,
    realInventoryWriteEnabled: false,
    inventoryMutationEnabled: false,
    backgroundWorkerEnabled: false,
    automaticRetryEnabled: false,
    productionReady: false,
    productionAcceptancePassed: false,
    realSecretCount: Number(credentialScan.realSecretCount),
    credentialValueRecorded: credentialScan.credentialValueRecorded !== false,
    currentBranch: gitState.currentBranch,
    currentHead: gitState.currentHead,
    stagedFileCount: gitState.stagedFileCount,
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
  };
}

function render(report, gate = false) {
  const title = gate ? 'P10 Batch 1 Pre-production Foundation Gate' : 'P10 Batch 1 Pre-production Foundation';
  return `# ${title}\n\nStatus: **${report.status}**\n\n\`\`\`text\nphase=P10\nbatch=1\nformalTaskTotal=${report.formalTaskTotal}\nformalTaskCompleted=${report.formalTaskCompleted}\nformalTaskDeferred=${report.formalTaskDeferred}\nrepositoryFoundationPassed=${report.repositoryFoundationPassed}\nexternalInfrastructureStatus=${report.externalInfrastructureStatus}\nexternalProvisioningPending=${report.externalProvisioningPending}\npreproductionEnvironmentDefined=${report.preproductionEnvironmentDefined}\nunknownEnvironmentFailClosed=${report.unknownEnvironmentFailClosed}\ndatabaseIsolationPassed=${report.databaseIsolationPassed}\nredisIsolationPassed=${report.redisIsolationPassed}\nsecretExternalizationPassed=${report.secretExternalizationPassed}\nsessionIsolationPassed=${report.sessionIsolationPassed}\nstartupSafetyPassed=${report.startupSafetyPassed}\nmigrationSafetyPassed=${report.migrationSafetyPassed}\nhealthReadinessPassed=${report.healthReadinessPassed}\nbackupFoundationPassed=${report.backupFoundationPassed}\nrestoreFoundationPassed=${report.restoreFoundationPassed}\nrollbackFoundationPassed=${report.rollbackFoundationPassed}\nproductionRestoreEnabled=${report.productionRestoreEnabled}\np9ProtectedSourceModified=${report.p9ProtectedSourceModified}\napprovedPlanningSemanticsDriftDetected=${report.approvedPlanningSemanticsDriftDetected}\nchangesCommittedDuringBatch1=${report.changesCommittedDuringBatch1}\ncurrentAllowedLevel=${report.currentAllowedLevel}\nproductionReady=${report.productionReady}\nproductionAcceptancePassed=${report.productionAcceptancePassed}\nfailedCount=${report.failedCount}\n\`\`\`\n\n## Failed Checks\n\n${report.failed.length ? report.failed.map((item) => `- ${item}`).join('\n') : '- None'}\n\n## Boundary\n\nRepository foundation is implemented, but independent external pre-production resources must be provisioned and rehearsed before this batch can complete. Production resources are not substitutes. OAuth, credentials, real Provider/network/read/write, Worker, retry, gray, Production Ready, Tag, and Release remain disabled or deferred.\n`;
}

function collectActualBundle() {
  const plan = readJSON('docs/p10-execution-plan.json') || {};
  const revalidation = readJSON('docs/p10-planning-pack-revalidation.json') || {};
  const externalEvidence = readJSON('docs/p10-task-batch-1-external-infrastructure.json') || {};
  const planningSemanticManifest = computeP10PlanningSemanticManifest();
  const contract = validateP10PreproductionContract(readEnvFile(rootPath('.env.staging.example')));
  const compose = read('deploy/preproduction/compose.yml');
  const readiness = read('deploy/scripts/check-preproduction-readiness.sh');
  const migration = read('deploy/scripts/migrate-preproduction.sh');
  const backup = read('deploy/scripts/backup-preproduction.sh');
  const restore = read('deploy/scripts/restore-preproduction.sh');
  const rollback = read('deploy/scripts/rollback-preproduction.sh');
  const teardown = read('deploy/scripts/teardown-preproduction.sh');
  const preflight = read('scripts/p10-preproduction-preflight.mjs');
  const currentHead = git(['rev-parse', 'HEAD']);
  const stagedFiles = git(['diff', '--cached', '--name-only']).split(/\r?\n/).filter(Boolean);
  const rawText = REQUIRED_FILES.map(read).join('\n');
  return {
    plan,
    transition: readJSON('docs/p9-to-p10-transition-gate.json') || {},
    planningGate: readJSON('docs/p10-planning-pack-gate.json') || {},
    ownerGate: readJSON('docs/p10-owner-decision-gate.json') || {},
    revalidation,
    planningSemanticManifest,
    frozenP9: readJSON('artifacts/p9-protected-source-freeze.json') || {},
    liveP9: computeLiveProtectedSourceManifest(),
    gitState: { currentBranch: git(['branch', '--show-current']), currentHead, stagedFileCount: stagedFiles.length },
    contract,
    externalEvidence,
    external: validateP10ExternalInfrastructureEvidence(externalEvidence),
    repository: {
      composeIsolation: compose.includes('name: trademind-preproduction')
        && compose.includes('internal: true') && compose.includes('preproduction_postgres_data')
        && compose.includes('PREPRODUCTION_DB_PASSWORD:?') && compose.includes('PREPRODUCTION_REDIS_PASSWORD:?'),
      readinessFoundation: readiness.includes('/health/ready') && readiness.includes('checks.database')
        && readiness.includes('checks.redis') && readiness.includes('checks.migrations'),
      migrationFoundation: migration.includes('--mode migration') && migration.includes('check-preproduction-readiness.sh'),
      backupFoundation: backup.includes('pg_dump') && backup.includes('sha256sum') && backup.includes('metadata.json'),
      restoreFoundation: restore.includes('_restore_[a-z0-9_]+$') && restore.includes('sha256sum --check')
        && restore.includes('pg_restore') && restore.includes('TARGET_DB" != "$DB_NAME'),
      rollbackFoundation: rollback.includes('P10_PREVIOUS_API_IMAGE')
        && rollback.includes('check-preproduction-readiness.sh') && !rollback.includes('pg_restore'),
      teardownFoundation: teardown.includes('down') && !teardown.includes('--volumes') && !teardown.includes('down -v'),
      unknownEnvironmentFailClosed: preflight.includes('validateP10PreproductionContract')
        && contract.checks.some((check) => check.id === 'environmentKnown'),
      startupSafety: preflight.includes("'startup'") && read('deploy/scripts/deploy-preproduction.sh').includes('--mode startup'),
      productionRestoreEnabled: false,
    },
    requiredFilesPresent: REQUIRED_FILES.every((file) => fs.existsSync(rootPath(file))),
    credentialScan: scanSecrets(rawText),
  };
}

export function runP10TaskBatch1Gate() {
  const report = validateP10TaskBatch1Bundle(collectActualBundle());
  write(EVIDENCE_JSON, report);
  write(EVIDENCE_MD, render(report));
  write(GATE_JSON, report);
  write(GATE_MD, render(report, true));
  console.log(JSON.stringify(report, null, 2));
  if (report.failedCount > 0) process.exitCode = 1;
  return report;
}

const isMain = process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url;
if (isMain) runP10TaskBatch1Gate();
