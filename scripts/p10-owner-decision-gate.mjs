import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { computeLiveProtectedSourceManifest } from './p9-protected-source-freeze.mjs';
import { computeP10PlanningSemanticManifest } from './p10-planning-semantic-manifest.mjs';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const OWNER_JSON = 'docs/p10-owner-scope-decision.json';
const BOUNDARY_JSON = 'docs/p10-production-boundary.json';
const PLAN_JSON = 'docs/p10-execution-plan.json';
const CRITERIA_JSON = 'docs/p10-acceptance-criteria.json';
const PACK_JSON = 'docs/p10-planning-pack.json';
const REVALIDATION_JSON = 'docs/p10-planning-pack-revalidation.json';
const TRANSITION_JSON = 'docs/p9-to-p10-transition-gate.json';
const FREEZE_JSON = 'artifacts/p9-protected-source-freeze.json';
export const P10_OWNER_DECISION_GATE_JSON = 'docs/p10-owner-decision-gate.json';
export const P10_OWNER_DECISION_GATE_MD = 'docs/P10_OWNER_DECISION_GATE.md';

const REQUIRED_FILES = [
  'docs/P10_OWNER_SCOPE_DECISION.md', OWNER_JSON,
  'docs/P10_PRODUCTION_BOUNDARY.md', BOUNDARY_JSON,
  'docs/P10_EXECUTION_PLAN.md', PLAN_JSON,
  'docs/P10_ACCEPTANCE_CRITERIA.md', CRITERIA_JSON,
  'docs/P10_PLANNING_PACK_REVALIDATION.md', REVALIDATION_JSON,
];
const EXPECTED_DECISION_IDS = Array.from({ length: 15 }, (_, index) => `D${String(index + 1).padStart(2, '0')}`);
const EXPECTED_WORKSTREAM_IDS = Array.from({ length: 11 }, (_, index) => `P10-W${index + 1}`);
const EXPECTED_BATCH_IDS = Array.from({ length: 9 }, (_, index) => `P10-B${index + 1}`);
const CREDENTIAL_KEYS = /^(accessToken|refreshToken|authorization|cookie|clientSecret|appSecret|apiKey|privateKey|password)$/i;

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
function exactMembers(actual = [], expected = []) {
  return actual.length === expected.length && expected.every((value) => actual.includes(value));
}
function allFalse(values) { return values.every((value) => value === false); }

function countCredentialValues(value) {
  if (Array.isArray(value)) return value.reduce((total, item) => total + countCredentialValues(item), 0);
  if (!value || typeof value !== 'object') return 0;
  return Object.entries(value).reduce((total, [key, item]) => {
    const recorded = CREDENTIAL_KEYS.test(key) && typeof item === 'string' && item.trim().length > 0 ? 1 : 0;
    return total + recorded + countCredentialValues(item);
  }, 0);
}

export function scanOwnerDecisionContent(documents = [], rawText = '') {
  const structuredCount = documents.reduce((total, document) => total + countCredentialValues(document), 0);
  const patterns = [
    /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/g,
    /\bBearer\s+[A-Za-z0-9._~-]{20,}/g,
    /\b(?:postgresql|postgres|mysql):\/\/[^\s:@/]+:[^\s@/]+@/g,
  ];
  const textCount = patterns.reduce((total, pattern) => total + (rawText.match(pattern)?.length || 0), 0);
  const realSecretCount = structuredCount + textCount;
  return { realSecretCount, credentialValueRecorded: realSecretCount > 0 };
}

export function validateP10OwnerDecisionBundle({
  owner = {}, boundary = {}, plan = {}, criteria = {}, pack = {}, transition = {}, freeze = {},
  revalidation = {}, planningSemanticManifest = {}, liveProtectedSourceManifest = {}, gitState = {},
  requiredFilesPresent = true, credentialScan = {},
} = {}) {
  const decisions = owner.decisions || [];
  const implementation = owner.implementationApprovals || {};
  const provider = owner.providerImplementation || {};
  const shopScope = owner.shopScope || {};
  const futureWrite = owner.futureWritePolicy || {};
  const credentialSecurity = owner.credentialSecurity || {};
  const gray = owner.grayScope || {};
  const automation = owner.initialAutomation || {};
  const recovery = owner.environmentAndRecovery || {};
  const performance = owner.performanceAcceptance || {};
  const ownerRuntime = owner.runtimeBoundary || {};
  const boundaryImplementation = boundary.implementationApprovals || {};
  const boundaryRuntime = boundary.runtimeBoundary || {};
  const boundaryGray = boundary.initialGrayScope || {};
  const workstreams = plan.workstreams || [];
  const batches = plan.batches || [];
  const w1 = workstreams.find((item) => item.workstreamId === 'P10-W1');
  const w11 = workstreams.find((item) => item.workstreamId === 'P10-W11');
  const batch1 = batches.find((item) => item.batchId === 'P10-B1');
  const acceptance = criteria.acceptanceCriteria || [];
  const writeCriterion = acceptance.find((item) => item.id === 'P10-AC-28');
  const p9ProtectedSourceDriftDetected = !freeze.sha256
    || !liveProtectedSourceManifest.sha256
    || freeze.sha256 !== liveProtectedSourceManifest.sha256
    || freeze.gitHead !== gitState.currentHead;
  const p9ProtectedSourceModified = transition.p9ProtectedChangedFileCount !== 0
    || transition.dirtyProtectedSourceDriftDetected !== false
    || p9ProtectedSourceDriftDetected;
  const realSecretCount = Number(credentialScan.realSecretCount ?? -1);
  const credentialValueRecorded = credentialScan.credentialValueRecorded !== false;
  const changesCommittedDuringCurrentRun = gitState.currentHead !== revalidation.currentRunBaseHead;
  const planningSemanticManifestMatches = Boolean(planningSemanticManifest.sha256)
    && planningSemanticManifest.sha256 === revalidation.planningSemanticManifestSha256
    && planningSemanticManifest.fileCount === revalidation.planningSemanticFileCount;

  const checks = [
    ['requiredFilesPresent', requiredFilesPresent],
    ['ownerDecisionCount', owner.ownerDecisionCount === 15 && decisions.length === 15 && exactMembers(decisions.map((item) => item.decisionId), EXPECTED_DECISION_IDS)],
    ['ownerApprovedDecisionCount', owner.ownerApprovedDecisionCount === 15 && decisions.every((item) => item.status === 'approved' && item.ownerApproved === true)],
    ['ownerApprovalPending', owner.ownerApprovalPending === false && owner.ownerDecisionStatus === 'approved' && owner.p10OwnerDecisionApproved === true],
    ['productionBoundaryApproved', boundary.productionBoundaryApproved === true && boundary.status === 'approved' && boundary.currentAllowedLevel === 'L0'],
    ['executionPlanFinalized', plan.executionPlanFinalized === true && plan.p10ExecutionPlanFinalized === true && plan.planStatus === 'finalized'],
    ['acceptanceCriteriaFinalized', criteria.acceptanceCriteriaFinalized === true && criteria.status === 'finalized' && criteria.allCriteriaApproved === true && criteria.allCriteriaPassed === false && acceptance.length >= 25],
    ['realProviderApprovedForImplementation', implementation.realProviderApprovedForImplementation === true && boundaryImplementation.realProviderApprovedForImplementation === true],
    ['realReadApprovedForImplementation', implementation.realReadApprovedForImplementation === true && boundaryImplementation.realReadApprovedForImplementation === true],
    ['providerImplementationBoundary', provider.provider === 'douyin' && provider.existingPortRequired === 'InventoryProviderPort' && provider.secondInventorySyncDomainAllowed === false && provider.firstStage === 'read_only' && exactMembers(provider.allowedOperations, ['GET', 'read']) && exactMembers(provider.forbiddenCapabilities, ['inventory_mutation', 'publish', 'listing', 'write'])],
    ['shopScopeBoundary', exactMembers(shopScope.stages, ['preproduction_test_shop', 'one_allowlisted_production_shop']) && shopScope.productionShopLimit === 1 && shopScope.multipleProductionShopsAllowed === false && shopScope.generalAvailabilityAllowed === false],
    ['realInventoryWriteApproved', implementation.realInventoryWriteApproved === false && implementation.inventoryMutationApproved === false && boundaryImplementation.realInventoryWriteApproved === false && boundaryImplementation.inventoryMutationApproved === false],
    ['futureWritePolicy', futureWrite.manualSingleConfirmationRequired === true && exactMembers(futureWrite.requiredControls, ['explicit_user_confirmation', 'idempotency', 'expected_revision', 'tenant_allowlist', 'shop_allowlist', 'audit', 'write_kill_switch']) && allFalse([futureWrite.automaticWriteAllowed, futureWrite.bulkAutomaticMutationAllowed, futureWrite.silentRetryWriteAllowed])],
    ['backgroundWorkerApproved', implementation.backgroundWorkerApproved === false && boundaryImplementation.backgroundWorkerApproved === false],
    ['automaticRetryApproved', implementation.automaticRetryApproved === false && boundaryImplementation.automaticRetryApproved === false],
    ['initialAutomationBoundary', automation.triggerMode === 'manual' && automation.rerunMode === 'manual' && automation.newBusinessRunCreatedAutomatically === false],
    ['initialGrayTenantLimit', gray.initialGrayTenantLimit === 1 && boundaryGray.initialGrayTenantLimit === 1],
    ['initialGrayShopLimit', gray.initialGrayShopLimit === 1 && boundaryGray.initialGrayShopLimit === 1],
    ['initialGraySkuLimit', gray.initialGraySkuLimit === 100 && boundaryGray.initialGraySkuLimit === 100],
    ['independentPreproductionRequired', recovery.independentPreproductionRequired === true],
    ['credentialSecurity', ['backendOnly','encryptedAtRest','managedKeyRequired','rotatable','revocable','audited','redacted'].every((key) => credentialSecurity[key] === true)],
    ['rpoMinutesMax', recovery.rpoMinutesMax === 15],
    ['rtoMinutesMax', recovery.rtoMinutesMax === 60],
    ['performanceAcceptance', performance.dedicatedHostRequired === true && performance.repeatRunsMinimum >= 3 && performance.sloFreezeRequiredBefore === 'G1'],
    ['repositoryBaselineDisposition', owner.repositoryBaselineDisposition === 'option_b' && plan.repositoryBaseline?.disposition === 'option_b' && plan.repositoryBaseline?.silentlyWaived === false && plan.repositoryBaseline?.newOrGrowthViolationAllowed === false],
    ['grayApprovalMode', owner.grayApprovalMode === 'owner_and_technical_lead' && exactMembers(owner.grayApprovalRequiredApprovers, ['owner', 'technical_lead'])],
    ['productionFinalApprover', owner.productionFinalApprover === 'owner' && exactMembers(owner.productionFinalApprovalPrerequisites, ['technical_sign_off', 'operations_security_sign_off', 'final_production_gate_passed'])],
    ['workstreamFinalization', workstreams.length === 11 && exactMembers(workstreams.map((item) => item.workstreamId), EXPECTED_WORKSTREAM_IDS) && w1?.status === 'completed' && workstreams.filter((item) => /^P10-W(?:[2-9]|10)$/.test(item.workstreamId)).every((item) => item.status === 'not_started')],
    ['conditionalWriteDeferred', w11?.conditional === true && w11?.status === 'deferred_pending_separate_owner_approval' && w11?.requiredForInitialProductionReady === false],
    ['batchPlanFinalized', batches.length === 9 && exactMembers(batches.map((item) => item.batchId), EXPECTED_BATCH_IDS) && batches.every((item) => ['taskIds','entryCriteria','dependencies','implementationScope','forbiddenScope','evidence','tests','exitCriteria','rollback'].every((key) => Array.isArray(item[key]) && item[key].length > 0) && typeof item.gate === 'string' && item.gate.length > 0 && typeof item.ownerApprovalRequirement === 'string' && item.ownerApprovalRequirement.length > 0)],
    ['batch1ReadyToStart', batch1?.status === 'ready_to_start' && batch1?.implementationStarted === false && batches.slice(1).every((item) => item.status === 'not_started' && item.implementationStarted === false)],
    ['conditionalWriteAcceptance', writeCriterion?.conditional === true && writeCriterion?.requiredForInitialProductionReady === false && criteria.initialProductionWriteRequirement?.realInventoryWriteRequired === false && criteria.initialProductionWriteRequirement?.unauthorizedWriteImpossibleRequired === true && criteria.initialProductionWriteRequirement?.inventoryMutationEnabled === false],
    ['p10ImplementationStarted', owner.p10ImplementationStarted === false && plan.p10ImplementationStarted === false && pack.p10ImplementationStarted === false],
    ['runtimeCapabilitiesDisabled', allFalse([ownerRuntime.realPlatformNetworkEnabled, ownerRuntime.realCredentialsEnabled, ownerRuntime.inventoryMutationEnabled, ownerRuntime.backgroundWorkerEnabled, ownerRuntime.automaticBusinessRetryEnabled, boundaryRuntime.realPlatformNetworkEnabled, boundaryRuntime.realCredentialsEnabled, boundaryRuntime.inventoryMutationEnabled, boundaryRuntime.backgroundWorkerEnabled, boundaryRuntime.automaticRetryEnabled, boundaryRuntime.automaticBusinessRetryEnabled])],
    ['productionReady', owner.productionReady === false && boundary.productionReady === false && plan.productionReady === false && criteria.productionReady === false],
    ['productionAcceptancePassed', owner.productionAcceptancePassed === false && boundary.productionAcceptancePassed === false && plan.productionAcceptancePassed === false && criteria.productionAcceptancePassed === false],
    ['p9ClosureReuseEligible', transition.status === 'passed' && transition.failedCount === 0 && transition.p9ClosureReuseEligible === true],
    ['p10PlanningEntryAllowed', transition.p10PlanningEntryAllowed === true],
    ['p9ProtectedSourceModified', p9ProtectedSourceModified === false],
    ['tagCreated', owner.tagCreated === false && gitState.tagCreated === false],
    ['releaseCreated', owner.releaseCreated === false],
    ['realSecretCount', realSecretCount === 0],
    ['credentialValueRecorded', credentialValueRecorded === false && owner.credentialValueRecorded === false],
    ['currentBranch', gitState.currentBranch === 'dev'],
    ['planningSemanticRevalidationPassed', revalidation.status === 'passed'
      && revalidation.planningSemanticRevalidationPassed === true],
    ['planningPackCurrentHeadValid', revalidation.planningPackCurrentHeadValid === true
      && revalidation.currentPlanningValidationHead === gitState.currentHead],
    ['planningSemanticManifest', planningSemanticManifestMatches],
    ['changesCommittedDuringCurrentRun', revalidation.changesCommittedDuringCurrentRun === false
      && changesCommittedDuringCurrentRun === false],
    ['changesCommitted', changesCommittedDuringCurrentRun === false],
    ['stagedFileCount', gitState.stagedFileCount === 0],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    phase: 'P10', gate: 'P10-OWNER-DECISION', status: failed.length === 0 ? 'passed' : 'failed',
    checkedAt: new Date().toISOString(), failed, failedCount: failed.length,
    ownerDecisionCount: decisions.length, ownerApprovedDecisionCount: decisions.filter((item) => item.status === 'approved' && item.ownerApproved === true).length,
    ownerApprovalPending: owner.ownerApprovalPending !== false,
    p10PlanningPackPrepared: pack.p10PlanningPackPrepared === true,
    p10OwnerDecisionApproved: owner.p10OwnerDecisionApproved === true,
    p10ExecutionPlanFinalized: plan.p10ExecutionPlanFinalized === true,
    p10ImplementationStarted: owner.p10ImplementationStarted === true || plan.p10ImplementationStarted === true,
    currentAllowedLevel: boundary.currentAllowedLevel,
    realProviderApprovedForImplementation: implementation.realProviderApprovedForImplementation === true,
    realReadApprovedForImplementation: implementation.realReadApprovedForImplementation === true,
    realInventoryWriteApproved: implementation.realInventoryWriteApproved === true,
    backgroundWorkerApproved: implementation.backgroundWorkerApproved === true,
    automaticRetryApproved: implementation.automaticRetryApproved === true,
    initialGrayTenantLimit: gray.initialGrayTenantLimit,
    initialGrayShopLimit: gray.initialGrayShopLimit,
    initialGraySkuLimit: gray.initialGraySkuLimit,
    independentPreproductionRequired: recovery.independentPreproductionRequired === true,
    rpoMinutesMax: recovery.rpoMinutesMax,
    rtoMinutesMax: recovery.rtoMinutesMax,
    repositoryBaselineDisposition: owner.repositoryBaselineDisposition,
    grayApprovalMode: owner.grayApprovalMode,
    productionFinalApprover: owner.productionFinalApprover,
    p9ClosureReuseEligible: transition.p9ClosureReuseEligible === true,
    p10PlanningEntryAllowed: transition.p10PlanningEntryAllowed === true,
    p9ProtectedSourceModified,
    productionReady: owner.productionReady === true,
    productionAcceptancePassed: owner.productionAcceptancePassed === true,
    realPlatformNetworkEnabled: ownerRuntime.realPlatformNetworkEnabled === true,
    realCredentialsEnabled: ownerRuntime.realCredentialsEnabled === true,
    inventoryMutationEnabled: ownerRuntime.inventoryMutationEnabled === true,
    tagCreated: owner.tagCreated === true || gitState.tagCreated === true,
    releaseCreated: owner.releaseCreated === true,
    realSecretCount,
    credentialValueRecorded,
    currentBranch: gitState.currentBranch,
    currentPlanningValidationHead: revalidation.currentPlanningValidationHead || '',
    planningSemanticRevalidationPassed: revalidation.planningSemanticRevalidationPassed === true,
    planningPackCurrentHeadValid: revalidation.planningPackCurrentHeadValid === true
      && revalidation.currentPlanningValidationHead === gitState.currentHead,
    planningSemanticManifestSha256: planningSemanticManifest.sha256 || '',
    planningSemanticManifestMatches,
    changesCommittedDuringCurrentRun,
    changesCommitted: changesCommittedDuringCurrentRun,
    stagedFileCount: gitState.stagedFileCount,
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
    nextAction: owner.nextAction,
  };
}

function renderMarkdown(report) {
  return `# P10 Owner Decision Gate\n\nStatus: **${report.status}**\n\n\`\`\`text\nownerDecisionCount=${report.ownerDecisionCount}\nownerApprovedDecisionCount=${report.ownerApprovedDecisionCount}\nownerApprovalPending=${report.ownerApprovalPending}\np10PlanningPackPrepared=${report.p10PlanningPackPrepared}\np10OwnerDecisionApproved=${report.p10OwnerDecisionApproved}\np10ExecutionPlanFinalized=${report.p10ExecutionPlanFinalized}\np10ImplementationStarted=${report.p10ImplementationStarted}\ncurrentAllowedLevel=${report.currentAllowedLevel}\nrealProviderApprovedForImplementation=${report.realProviderApprovedForImplementation}\nrealReadApprovedForImplementation=${report.realReadApprovedForImplementation}\nrealInventoryWriteApproved=${report.realInventoryWriteApproved}\nbackgroundWorkerApproved=${report.backgroundWorkerApproved}\nautomaticRetryApproved=${report.automaticRetryApproved}\ninitialGrayTenantLimit=${report.initialGrayTenantLimit}\ninitialGrayShopLimit=${report.initialGrayShopLimit}\ninitialGraySkuLimit=${report.initialGraySkuLimit}\nindependentPreproductionRequired=${report.independentPreproductionRequired}\nrpoMinutesMax=${report.rpoMinutesMax}\nrtoMinutesMax=${report.rtoMinutesMax}\nrepositoryBaselineDisposition=${report.repositoryBaselineDisposition}\ngrayApprovalMode=${report.grayApprovalMode}\nproductionFinalApprover=${report.productionFinalApprover}\np9ClosureReuseEligible=${report.p9ClosureReuseEligible}\np10PlanningEntryAllowed=${report.p10PlanningEntryAllowed}\np9ProtectedSourceModified=${report.p9ProtectedSourceModified}\ncurrentPlanningValidationHead=${report.currentPlanningValidationHead}\nplanningSemanticRevalidationPassed=${report.planningSemanticRevalidationPassed}\nplanningPackCurrentHeadValid=${report.planningPackCurrentHeadValid}\nplanningSemanticManifestSha256=${report.planningSemanticManifestSha256}\nplanningSemanticManifestMatches=${report.planningSemanticManifestMatches}\nchangesCommittedDuringCurrentRun=${report.changesCommittedDuringCurrentRun}\nproductionReady=${report.productionReady}\nproductionAcceptancePassed=${report.productionAcceptancePassed}\nrealPlatformNetworkEnabled=${report.realPlatformNetworkEnabled}\nrealCredentialsEnabled=${report.realCredentialsEnabled}\ninventoryMutationEnabled=${report.inventoryMutationEnabled}\ntagCreated=${report.tagCreated}\nreleaseCreated=${report.releaseCreated}\nrealSecretCount=${report.realSecretCount}\ncredentialValueRecorded=${report.credentialValueRecorded}\ncurrentBranch=${report.currentBranch}\nchangesCommitted=${report.changesCommitted}\nstagedFileCount=${report.stagedFileCount}\nfailedCount=${report.failedCount}\n\`\`\`\n\n## Failed Checks\n\n${report.failed.length ? report.failed.map((item) => `- ${item}`).join('\n') : '- None'}\n\n## Boundary\n\nThis gate approves the P10 implementation plan only. It does not start implementation or enable OAuth, credentials, a real Provider, platform network access, inventory reads or writes, Worker, automatic retry, gray, Tag, Release, or Production Ready.\n\nNext: **${report.nextAction}**.\n`;
}

function collectActualBundle() {
  const documents = [OWNER_JSON, BOUNDARY_JSON, PLAN_JSON, CRITERIA_JSON, PACK_JSON, REVALIDATION_JSON].map(readJSON);
  const rawText = REQUIRED_FILES.map(read).join('\n');
  const currentHead = git(['rev-parse', 'HEAD']);
  const stagedFiles = git(['diff', '--cached', '--name-only']).split(/\r?\n/).filter(Boolean);
  const tagNames = git(['tag', '--points-at', 'HEAD']).split(/\r?\n/).filter(Boolean);
  return {
    owner: documents[0] || {}, boundary: documents[1] || {}, plan: documents[2] || {}, criteria: documents[3] || {}, pack: documents[4] || {},
    revalidation: documents[5] || {}, planningSemanticManifest: computeP10PlanningSemanticManifest(),
    transition: readJSON(TRANSITION_JSON) || {}, freeze: readJSON(FREEZE_JSON) || {}, liveProtectedSourceManifest: computeLiveProtectedSourceManifest(),
    gitState: { currentBranch: git(['branch', '--show-current']), currentHead, stagedFileCount: stagedFiles.length, tagCreated: tagNames.length > 0 },
    requiredFilesPresent: REQUIRED_FILES.every((file) => fs.existsSync(rootPath(file))),
    credentialScan: scanOwnerDecisionContent(documents, rawText),
  };
}

export function runP10OwnerDecisionGate() {
  const report = validateP10OwnerDecisionBundle(collectActualBundle());
  write(P10_OWNER_DECISION_GATE_JSON, report);
  write(P10_OWNER_DECISION_GATE_MD, renderMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.failedCount > 0) process.exitCode = 1;
  return report;
}

const isMain = process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url;
if (isMain) runP10OwnerDecisionGate();
