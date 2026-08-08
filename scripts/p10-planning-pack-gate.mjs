import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { computeLiveProtectedSourceManifest } from './p9-protected-source-freeze.mjs';
import { computeP10PlanningSemanticManifest } from './p10-planning-semantic-manifest.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const TRANSITION_JSON = 'docs/p9-to-p10-transition-gate.json';
const FREEZE_JSON = 'artifacts/p9-protected-source-freeze.json';
const PROPOSAL_JSON = 'docs/p10-owner-decision-proposal.json';
const BOUNDARY_JSON = 'docs/p10-production-boundary.json';
const PLAN_JSON = 'docs/p10-execution-plan-draft.json';
const CRITERIA_JSON = 'docs/p10-acceptance-criteria-draft.json';
const RISK_JSON = 'docs/p10-risk-register.json';
const PACK_JSON = 'docs/p10-planning-pack.json';
const REVALIDATION_JSON = 'docs/p10-planning-pack-revalidation.json';
export const P10_PLANNING_PACK_GATE_JSON = 'docs/p10-planning-pack-gate.json';
export const P10_PLANNING_PACK_GATE_MD = 'docs/P10_PLANNING_PACK_GATE.md';

const REQUIRED_FILES = [
  'docs/P10_OWNER_DECISION_PROPOSAL.md',
  PROPOSAL_JSON,
  'docs/P10_PRODUCTION_BOUNDARY.md',
  BOUNDARY_JSON,
  'docs/P10_EXECUTION_PLAN_DRAFT.md',
  PLAN_JSON,
  'docs/P10_ACCEPTANCE_CRITERIA_DRAFT.md',
  CRITERIA_JSON,
  'docs/P10_RISK_REGISTER.md',
  RISK_JSON,
  'docs/P10_PLANNING_PACK.md',
  PACK_JSON,
  'docs/P10_PLANNING_PACK_REVALIDATION.md',
  REVALIDATION_JSON,
];

function rootPath(relativePath) { return path.join(REPO_ROOT, relativePath); }
function read(relativePath) { try { return fs.readFileSync(rootPath(relativePath), 'utf8'); } catch { return ''; } }
function readJSON(relativePath) { try { return JSON.parse(read(relativePath)); } catch { return null; } }
function write(relativePath, value) {
  fs.mkdirSync(path.dirname(rootPath(relativePath)), { recursive: true });
  const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  fs.writeFileSync(rootPath(relativePath), content.endsWith('\n') ? content : `${content}\n`, 'utf8');
}
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}
function gitExitZero(args) {
  try { execFileSync('git', args, { cwd: REPO_ROOT, stdio: 'ignore' }); return true; } catch { return false; }
}
function allFalse(values) { return values.every((value) => value === false); }
function includesAll(values = [], required = []) { return required.every((value) => values.includes(value)); }

const CREDENTIAL_KEYS = /^(accessToken|refreshToken|authorization|cookie|clientSecret|appSecret|apiKey|privateKey|password)$/i;

function countCredentialValues(value) {
  if (Array.isArray(value)) return value.reduce((total, item) => total + countCredentialValues(item), 0);
  if (!value || typeof value !== 'object') return 0;
  return Object.entries(value).reduce((total, [key, item]) => {
    const recorded = CREDENTIAL_KEYS.test(key) && typeof item === 'string' && item.trim().length > 0 ? 1 : 0;
    return total + recorded + countCredentialValues(item);
  }, 0);
}

export function scanPlanningContent(documents = [], rawText = '') {
  const structuredCount = documents.reduce((total, document) => total + countCredentialValues(document), 0);
  const highConfidencePatterns = [
    /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/g,
    /\bBearer\s+[A-Za-z0-9._~-]{20,}/g,
    /\b(?:postgresql|postgres|mysql):\/\/[^\s:@/]+:[^\s@/]+@/g,
  ];
  const textCount = highConfidencePatterns.reduce((total, pattern) => total + (rawText.match(pattern)?.length || 0), 0);
  const realSecretCount = structuredCount + textCount;
  return { realSecretCount, credentialValueRecorded: realSecretCount > 0 };
}

export function validateP10PlanningPackBundle({
  transition = {},
  freeze = {},
  liveProtectedSourceManifest = {},
  proposal = {},
  boundary = {},
  plan = {},
  criteria = {},
  risks = {},
  pack = {},
  revalidation = {},
  planningSemanticManifest = {},
  gitState = {},
  requiredFilesPresent = true,
  credentialScan = {},
} = {}) {
  const decisions = proposal.decisions || [];
  const decisionIds = decisions.map((decision) => decision.decisionId);
  const expectedDecisionIds = Array.from({ length: 15 }, (_, index) => `D${String(index + 1).padStart(2, '0')}`);
  const decisionsPending = decisions.length === 15
    && includesAll(decisionIds, expectedDecisionIds)
    && proposal.recommendationOnly === true
    && proposal.approved === false
    && decisions.every((decision) => decision.ownerApprovalRequired === true
      && decision.recommendationOnly === true
      && decision.approved === false
      && decision.status === 'pending_owner_approval'
      && typeof decision.question === 'string'
      && decision.question.length > 0
      && Array.isArray(decision.options)
      && decision.options.length >= 2
      && typeof decision.recommendedOption === 'string'
      && decision.recommendedOption.length > 0
      && typeof decision.recommendationReason === 'string'
      && decision.recommendationReason.length > 0
      && typeof decision.risk === 'string'
      && Array.isArray(decision.dependencies));
  const approvals = boundary.approvals || {};
  const runtimeBoundary = boundary.runtimeBoundary || {};
  const workstreams = plan.workstreams || [];
  const w11 = workstreams.find((item) => item.workstreamId === 'P10-W11');
  const rollback = plan.rollbackPlan || {};
  const performance = plan.performanceClosurePlan || {};
  const performanceEvidence = performance.requiredEvidence || [];
  const repositoryBaseline = plan.repositoryBaseline || {};
  const p9ProtectedSourceDriftDetected = !freeze.sha256
    || !liveProtectedSourceManifest.sha256
    || freeze.sha256 !== liveProtectedSourceManifest.sha256
    || freeze.gitHead !== gitState.currentHead;
  const p9ProtectedSourceModified = transition.p9ProtectedChangedFileCount !== 0
    || transition.dirtyProtectedSourceDriftDetected !== false
    || p9ProtectedSourceDriftDetected;
  const realSecretCount = Number(credentialScan.realSecretCount ?? -1);
  const credentialValueRecorded = credentialScan.credentialValueRecorded !== false;
  const historicalPlanningCheckpoint = revalidation.initialPlanningCheckpoint || pack.baseCheckpoint || '';
  const currentRunBaseHead = revalidation.currentRunBaseHead || '';
  const changesCommittedDuringCurrentRun = gitState.currentHead !== currentRunBaseHead;
  const planningSemanticManifestMatches = Boolean(planningSemanticManifest.sha256)
    && planningSemanticManifest.sha256 === revalidation.planningSemanticManifestSha256
    && planningSemanticManifest.fileCount === revalidation.planningSemanticFileCount;

  const checks = [
    ['requiredFilesPresent', requiredFilesPresent],
    ['p9ToP10TransitionPassed', transition.status === 'passed' && transition.failedCount === 0],
    ['p9ClosureReuseEligible', transition.p9ClosureReuseEligible === true],
    ['p10PlanningEntryAllowed', transition.p10PlanningEntryAllowed === true],
    ['p9ProtectedSourceModified', p9ProtectedSourceModified === false && pack.p9ProtectedSourceModified === false && pack.p9ProtectedSourceModifiedDuringPlanning === false],
    ['p9ProtectedSourceDriftDetected', p9ProtectedSourceDriftDetected === false && pack.p9ProtectedSourceDriftDetected === false],
    ['p10PlanningPackPrepared', pack.p10PlanningPackPrepared === true && pack.planningStatus === 'prepared_for_owner_review'],
    ['p10ImplementationStarted', pack.p10ImplementationStarted === false],
    ['ownerDecisionProposalPresent', pack.ownerDecisionProposalPresent === true],
    ['ownerDecisionCount', proposal.ownerDecisionCount === 15 && decisionsPending],
    ['ownerApprovalPending', proposal.ownerApprovalPending === true && pack.ownerApprovalPending === true],
    ['unapprovedDecisionTreatedAsApproved', pack.unapprovedDecisionTreatedAsApproved === false],
    ['productionBoundaryPresent', pack.productionBoundaryPresent === true && boundary.currentAllowedLevel === 'L0' && boundary.levels?.length === 4],
    ['executionPlanDraftPresent', pack.executionPlanDraftPresent === true && workstreams.length === 11],
    ['acceptanceCriteriaDraftPresent', pack.acceptanceCriteriaDraftPresent === true && (criteria.acceptanceCriteria || []).length >= 24],
    ['riskRegisterPresent', pack.riskRegisterPresent === true && (risks.risks || []).length >= 16],
    ['realProviderApproved', pack.realProviderApproved === false && approvals.realProviderApproved === false],
    ['realInventoryReadApproved', pack.realInventoryReadApproved === false && approvals.realInventoryReadApproved === false],
    ['realInventoryWriteApproved', pack.realInventoryWriteApproved === false && approvals.realInventoryWriteApproved === false],
    ['backgroundWorkerApproved', pack.backgroundWorkerApproved === false && approvals.backgroundWorkerApproved === false],
    ['automaticRetryApproved', pack.automaticRetryApproved === false && approvals.automaticRetryApproved === false],
    ['runtimeProductionCapabilitiesDisabled', allFalse([runtimeBoundary.realPlatformNetworkEnabled, runtimeBoundary.realCredentialsEnabled, runtimeBoundary.inventoryMutationEnabled, runtimeBoundary.backgroundWorkerEnabled, runtimeBoundary.automaticRetryEnabled])],
    ['conditionalWriteBlocked', w11?.conditional === true && w11?.status === 'blocked_pending_owner_approval'],
    ['rollbackPlanPresent', rollback.postgresqlBackup === true && rollback.pointInTimeRecovery === true && rollback.restoreDrill === true && rollback.deploymentRollback === true && rollback.migrationRecovery === true && rollback.credentialRevocation === true],
    ['killSwitchPlanPresent', includesAll(plan.killSwitches, ['provider', 'tenant', 'shop', 'read', 'write'])],
    ['p7PerformanceClosurePlanPresent', performance.status === 'open_blocking_for_production'
      && performance.historicalFacts?.completedRunCount === 2
      && performance.historicalFacts?.expectedRunCount === 4
      && performance.historicalFacts?.failedCount === 8
      && performance.historicalFacts?.validForFormalPlan === false
      && performance.historicalWaiverIsProductionAcceptance === false
      && includesAll(performanceEvidence, ['P50, P95, and P99 latency', 'Throughput and concurrency', 'CPU and memory', 'Database connections', 'Provider rate-limit behavior'])],
    ['repositoryBaselineDispositionPending', pack.repositoryBaselineDisposition === 'pending_owner_approval' && repositoryBaseline.disposition === 'pending_owner_approval' && repositoryBaseline.silentlyWaived === false && repositoryBaseline.knownItems?.length === 3 && repositoryBaseline.options?.length === 3],
    ['productionReady', pack.productionReady === false && boundary.productionReady === false],
    ['productionAcceptancePassed', pack.productionAcceptancePassed === false && boundary.productionAcceptancePassed === false && criteria.productionAcceptancePassed === false],
    ['tagCreated', pack.tagCreated === false && gitState.tagCreated === false],
    ['releaseCreated', pack.releaseCreated === false],
    ['realSecretCount', realSecretCount === 0],
    ['credentialValueRecorded', credentialValueRecorded === false && pack.credentialValueRecorded === false],
    ['currentBranch', gitState.currentBranch === 'dev' && pack.baseBranch === 'dev'],
    ['historicalPlanningPackValid', pack.changesCommitted === false && historicalPlanningCheckpoint === pack.baseCheckpoint],
    ['historicalPlanningPackHeadIsAncestor', gitState.historicalPlanningPackHeadIsAncestor === true],
    ['planningSemanticRevalidationPassed', revalidation.status === 'passed'
      && revalidation.planningSemanticRevalidationPassed === true
      && typeof revalidation.planningSemanticsUnchanged === 'boolean'],
    ['planningPackCurrentHeadValid', revalidation.planningPackCurrentHeadValid === true
      && revalidation.currentPlanningValidationHead === gitState.currentHead],
    ['planningSemanticManifest', planningSemanticManifestMatches],
    ['changesCommittedDuringCurrentRun', revalidation.changesCommittedDuringCurrentRun === false
      && changesCommittedDuringCurrentRun === false],
    ['changesCommitted', pack.changesCommitted === false && changesCommittedDuringCurrentRun === false],
    ['stagedFileCount', gitState.stagedFileCount === 0 && pack.stagedFileCount === 0],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    phase: 'P10',
    gate: 'P10-PLANNING-PACK',
    status: failed.length === 0 ? 'passed' : 'failed',
    checkedAt: new Date().toISOString(),
    failed,
    failedCount: failed.length,
    currentBranch: gitState.currentBranch,
    currentHead: gitState.currentHead,
    stagedFileCount: gitState.stagedFileCount,
    historicalPlanningCheckpoint,
    historicalPlanningPackHeadIsAncestor: gitState.historicalPlanningPackHeadIsAncestor === true,
    currentPlanningValidationHead: revalidation.currentPlanningValidationHead || '',
    currentRunBaseHead,
    planningCheckpointAdvanced: gitState.currentHead !== pack.baseCheckpoint,
    planningSemanticsUnchanged: revalidation.planningSemanticsUnchanged === true,
    planningSemanticRevalidationPassed: revalidation.planningSemanticRevalidationPassed === true,
    planningPackCurrentHeadValid: revalidation.planningPackCurrentHeadValid === true
      && revalidation.currentPlanningValidationHead === gitState.currentHead,
    planningSemanticManifestSha256: planningSemanticManifest.sha256 || '',
    planningSemanticManifestMatches,
    changesCommittedDuringCurrentRun,
    changesCommitted: changesCommittedDuringCurrentRun,
    p9ToP10TransitionPassed: transition.status === 'passed' && transition.failedCount === 0,
    p9ClosureReuseEligible: transition.p9ClosureReuseEligible === true,
    p10PlanningEntryAllowed: transition.p10PlanningEntryAllowed === true,
    p9ProtectedSourceModified,
    p9ProtectedSourceDriftDetected,
    beforePlanningManifestSha256: freeze.sha256 || '',
    afterPlanningManifestSha256: liveProtectedSourceManifest.sha256 || '',
    p10PlanningPackPrepared: pack.p10PlanningPackPrepared === true,
    p10ImplementationStarted: pack.p10ImplementationStarted === true,
    ownerDecisionProposalPresent: pack.ownerDecisionProposalPresent === true,
    ownerDecisionCount: decisions.length,
    ownerApprovalPending: proposal.ownerApprovalPending === true,
    unapprovedDecisionTreatedAsApproved: !decisionsPending,
    productionBoundaryPresent: pack.productionBoundaryPresent === true,
    executionPlanDraftPresent: pack.executionPlanDraftPresent === true,
    acceptanceCriteriaDraftPresent: pack.acceptanceCriteriaDraftPresent === true,
    riskRegisterPresent: pack.riskRegisterPresent === true,
    realProviderApproved: pack.realProviderApproved === true,
    realInventoryReadApproved: pack.realInventoryReadApproved === true,
    realInventoryWriteApproved: pack.realInventoryWriteApproved === true,
    backgroundWorkerApproved: pack.backgroundWorkerApproved === true,
    automaticRetryApproved: pack.automaticRetryApproved === true,
    repositoryBaselineDisposition: pack.repositoryBaselineDisposition,
    productionReady: pack.productionReady === true,
    productionAcceptancePassed: pack.productionAcceptancePassed === true,
    tagCreated: pack.tagCreated === true || gitState.tagCreated === true,
    releaseCreated: pack.releaseCreated === true,
    realSecretCount,
    credentialValueRecorded,
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
  };
}

function renderMarkdown(report) {
  return `# P10 Planning Pack Gate\n\nStatus: **${report.status}**\n\n\`\`\`text\nphase=P10\np10PlanningPackPrepared=${report.p10PlanningPackPrepared}\np10ImplementationStarted=${report.p10ImplementationStarted}\np9ToP10TransitionPassed=${report.p9ToP10TransitionPassed}\np9ClosureReuseEligible=${report.p9ClosureReuseEligible}\np10PlanningEntryAllowed=${report.p10PlanningEntryAllowed}\np9ProtectedSourceModified=${report.p9ProtectedSourceModified}\np9ProtectedSourceDriftDetected=${report.p9ProtectedSourceDriftDetected}\nhistoricalPlanningCheckpoint=${report.historicalPlanningCheckpoint}\nhistoricalPlanningPackHeadIsAncestor=${report.historicalPlanningPackHeadIsAncestor}\ncurrentPlanningValidationHead=${report.currentPlanningValidationHead}\nplanningCheckpointAdvanced=${report.planningCheckpointAdvanced}\nplanningSemanticsUnchanged=${report.planningSemanticsUnchanged}\nplanningSemanticRevalidationPassed=${report.planningSemanticRevalidationPassed}\nplanningPackCurrentHeadValid=${report.planningPackCurrentHeadValid}\nplanningSemanticManifestSha256=${report.planningSemanticManifestSha256}\nplanningSemanticManifestMatches=${report.planningSemanticManifestMatches}\nchangesCommittedDuringCurrentRun=${report.changesCommittedDuringCurrentRun}\nownerDecisionCount=${report.ownerDecisionCount}\nownerApprovalPending=${report.ownerApprovalPending}\nrepositoryBaselineDisposition=${report.repositoryBaselineDisposition}\nproductionReady=${report.productionReady}\nproductionAcceptancePassed=${report.productionAcceptancePassed}\nrealSecretCount=${report.realSecretCount}\ncredentialValueRecorded=${report.credentialValueRecorded}\nfailedCount=${report.failedCount}\n\`\`\`\n\n## Failed Checks\n\n${report.failed.length ? report.failed.map((item) => `- ${item}`).join('\n') : '- None'}\n\n## Boundary\n\nThis gate validates planning evidence only. It does not approve or enable a real Provider, OAuth, platform network access, inventory reads or writes, Worker, automatic retry, gray release, Tag, Release, or Production Ready.\n`;
}

function collectActualBundle() {
  const documents = [PROPOSAL_JSON, BOUNDARY_JSON, PLAN_JSON, CRITERIA_JSON, RISK_JSON, PACK_JSON, REVALIDATION_JSON].map(readJSON);
  const rawText = REQUIRED_FILES.map(read).join('\n');
  const currentHead = git(['rev-parse', 'HEAD']);
  const stagedFiles = git(['diff', '--cached', '--name-only']).split(/\r?\n/).filter(Boolean);
  const tagNames = git(['tag', '--points-at', 'HEAD']).split(/\r?\n/).filter(Boolean);
  return {
    transition: readJSON(TRANSITION_JSON) || {},
    freeze: readJSON(FREEZE_JSON) || {},
    liveProtectedSourceManifest: computeLiveProtectedSourceManifest(),
    proposal: documents[0] || {},
    boundary: documents[1] || {},
    plan: documents[2] || {},
    criteria: documents[3] || {},
    risks: documents[4] || {},
    pack: documents[5] || {},
    revalidation: documents[6] || {},
    planningSemanticManifest: computeP10PlanningSemanticManifest(),
    gitState: {
      currentBranch: git(['branch', '--show-current']),
      currentHead,
      stagedFileCount: stagedFiles.length,
      tagCreated: tagNames.length > 0,
      historicalPlanningPackHeadIsAncestor: Boolean(documents[5]?.baseCheckpoint)
        && gitExitZero(['merge-base', '--is-ancestor', documents[5].baseCheckpoint, currentHead]),
    },
    requiredFilesPresent: REQUIRED_FILES.every((file) => fs.existsSync(rootPath(file))),
    credentialScan: scanPlanningContent(documents, rawText),
  };
}

export function runP10PlanningPackGate() {
  const report = validateP10PlanningPackBundle(collectActualBundle());
  write(P10_PLANNING_PACK_GATE_JSON, report);
  write(P10_PLANNING_PACK_GATE_MD, renderMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.failedCount > 0) process.exitCode = 1;
  return report;
}

const isMain = process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url;
if (isMain) runP10PlanningPackGate();
