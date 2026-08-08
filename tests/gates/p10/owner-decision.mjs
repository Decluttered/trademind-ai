import assert from 'node:assert/strict';
import { validateP10OwnerDecisionBundle } from '../../../scripts/p10-owner-decision-gate.mjs';

function validBundle() {
  const decisions = Array.from({ length: 15 }, (_, index) => ({
    decisionId: `D${String(index + 1).padStart(2, '0')}`,
    status: 'approved',
    ownerApproved: true,
  }));
  const batch = (index) => ({
    batchId: `P10-B${index}`,
    status: index === 1 ? 'ready_to_start' : 'not_started',
    implementationStarted: false,
    taskIds: ['task'], entryCriteria: ['entry'], dependencies: ['dependency'],
    implementationScope: ['scope'], forbiddenScope: ['forbidden'], evidence: ['evidence'],
    tests: ['test'], gate: 'gate', exitCriteria: ['exit'], rollback: ['rollback'],
    ownerApprovalRequirement: 'approval requirement',
  });
  return {
    owner: {
      ownerDecisionStatus: 'approved', ownerDecisionCount: 15, ownerApprovedDecisionCount: 15,
      ownerApprovalPending: false, p10OwnerDecisionApproved: true, decisions,
      implementationApprovals: {
        realProviderApprovedForImplementation: true, realReadApprovedForImplementation: true,
        realInventoryWriteApproved: false, inventoryMutationApproved: false,
        backgroundWorkerApproved: false, automaticRetryApproved: false,
      },
      providerImplementation: {
        provider: 'douyin', existingPortRequired: 'InventoryProviderPort', secondInventorySyncDomainAllowed: false,
        firstStage: 'read_only', allowedOperations: ['GET', 'read'], forbiddenCapabilities: ['inventory_mutation', 'publish', 'listing', 'write'],
      },
      shopScope: { stages: ['preproduction_test_shop', 'one_allowlisted_production_shop'], productionShopLimit: 1, multipleProductionShopsAllowed: false, generalAvailabilityAllowed: false },
      futureWritePolicy: {
        manualSingleConfirmationRequired: true,
        requiredControls: ['explicit_user_confirmation', 'idempotency', 'expected_revision', 'tenant_allowlist', 'shop_allowlist', 'audit', 'write_kill_switch'],
        automaticWriteAllowed: false, bulkAutomaticMutationAllowed: false, silentRetryWriteAllowed: false,
      },
      credentialSecurity: { backendOnly: true, encryptedAtRest: true, managedKeyRequired: true, rotatable: true, revocable: true, audited: true, redacted: true },
      grayScope: { initialGrayTenantLimit: 1, initialGrayShopLimit: 1, initialGraySkuLimit: 100 },
      initialAutomation: { triggerMode: 'manual', rerunMode: 'manual', newBusinessRunCreatedAutomatically: false },
      environmentAndRecovery: { independentPreproductionRequired: true, rpoMinutesMax: 15, rtoMinutesMax: 60 },
      performanceAcceptance: { dedicatedHostRequired: true, repeatRunsMinimum: 3, sloFreezeRequiredBefore: 'G1' },
      repositoryBaselineDisposition: 'option_b',
      grayApprovalMode: 'owner_and_technical_lead', grayApprovalRequiredApprovers: ['owner', 'technical_lead'],
      productionFinalApprover: 'owner', productionFinalApprovalPrerequisites: ['technical_sign_off', 'operations_security_sign_off', 'final_production_gate_passed'],
      runtimeBoundary: { realPlatformNetworkEnabled: false, realCredentialsEnabled: false, inventoryMutationEnabled: false, backgroundWorkerEnabled: false, automaticBusinessRetryEnabled: false },
      p10ImplementationStarted: false, productionReady: false, productionAcceptancePassed: false,
      tagCreated: false, releaseCreated: false, credentialValueRecorded: false,
    },
    boundary: {
      status: 'approved', productionBoundaryApproved: true, currentAllowedLevel: 'L0',
      implementationApprovals: {
        realProviderApprovedForImplementation: true, realReadApprovedForImplementation: true,
        realInventoryWriteApproved: false, inventoryMutationApproved: false,
        backgroundWorkerApproved: false, automaticRetryApproved: false,
      },
      initialGrayScope: { initialGrayTenantLimit: 1, initialGrayShopLimit: 1, initialGraySkuLimit: 100 },
      runtimeBoundary: { realPlatformNetworkEnabled: false, realCredentialsEnabled: false, inventoryMutationEnabled: false, backgroundWorkerEnabled: false, automaticRetryEnabled: false, automaticBusinessRetryEnabled: false },
      productionReady: false, productionAcceptancePassed: false,
    },
    plan: {
      planStatus: 'finalized', executionPlanFinalized: true, p10ExecutionPlanFinalized: true,
      p10ImplementationStarted: false,
      workstreams: [
        { workstreamId: 'P10-W1', status: 'completed' },
        ...Array.from({ length: 9 }, (_, index) => ({ workstreamId: `P10-W${index + 2}`, status: 'not_started' })),
        { workstreamId: 'P10-W11', status: 'deferred_pending_separate_owner_approval', conditional: true, requiredForInitialProductionReady: false },
      ],
      batches: Array.from({ length: 9 }, (_, index) => batch(index + 1)),
      repositoryBaseline: { disposition: 'option_b', silentlyWaived: false, newOrGrowthViolationAllowed: false },
      productionReady: false, productionAcceptancePassed: false,
    },
    criteria: {
      status: 'finalized', acceptanceCriteriaFinalized: true,
      acceptanceCriteria: [
        ...Array.from({ length: 27 }, (_, index) => ({ id: `P10-AC-${String(index + 1).padStart(2, '0')}`, requiredForInitialProductionReady: true })),
        { id: 'P10-AC-28', conditional: true, requiredForInitialProductionReady: false },
      ],
      initialProductionWriteRequirement: { realInventoryWriteRequired: false, unauthorizedWriteImpossibleRequired: true, inventoryMutationEnabled: false },
      allCriteriaApproved: true, allCriteriaPassed: false,
      productionReady: false, productionAcceptancePassed: false,
    },
    pack: { p10PlanningPackPrepared: true, p10ImplementationStarted: false, baseCheckpoint: 'base-head' },
    transition: { status: 'passed', failedCount: 0, p9ClosureReuseEligible: true, p10PlanningEntryAllowed: true, p9ProtectedChangedFileCount: 0, dirtyProtectedSourceDriftDetected: false },
    freeze: { sha256: 'protected', gitHead: 'base-head' },
    liveProtectedSourceManifest: { sha256: 'protected' },
    gitState: { currentBranch: 'dev', currentHead: 'base-head', stagedFileCount: 0, tagCreated: false },
    requiredFilesPresent: true,
    credentialScan: { realSecretCount: 0, credentialValueRecorded: false },
  };
}

const fixtures = [
  { id: 'OD-01', name: 'valid approved decisions pass', pass: true, mutate() {} },
  { id: 'OD-02', name: 'decision count below 15 fails', check: 'ownerDecisionCount', mutate(bundle) { bundle.owner.decisions.pop(); bundle.owner.ownerDecisionCount = 14; } },
  { id: 'OD-03', name: 'pending decision fails', check: 'ownerApprovedDecisionCount', mutate(bundle) { bundle.owner.decisions[0].status = 'pending_owner_approval'; } },
  { id: 'OD-04', name: 'real provider implementation not approved fails', check: 'realProviderApprovedForImplementation', mutate(bundle) { bundle.owner.implementationApprovals.realProviderApprovedForImplementation = false; } },
  { id: 'OD-05', name: 'real read implementation not approved fails', check: 'realReadApprovedForImplementation', mutate(bundle) { bundle.owner.implementationApprovals.realReadApprovedForImplementation = false; } },
  { id: 'OD-06', name: 'real write accidentally approved fails', check: 'realInventoryWriteApproved', mutate(bundle) { bundle.owner.implementationApprovals.realInventoryWriteApproved = true; } },
  { id: 'OD-07', name: 'worker approved fails', check: 'backgroundWorkerApproved', mutate(bundle) { bundle.owner.implementationApprovals.backgroundWorkerApproved = true; } },
  { id: 'OD-08', name: 'automatic retry approved fails', check: 'automaticRetryApproved', mutate(bundle) { bundle.owner.implementationApprovals.automaticRetryApproved = true; } },
  { id: 'OD-09', name: 'gray tenant above one fails', check: 'initialGrayTenantLimit', mutate(bundle) { bundle.owner.grayScope.initialGrayTenantLimit = 2; } },
  { id: 'OD-10', name: 'gray shop above one fails', check: 'initialGrayShopLimit', mutate(bundle) { bundle.owner.grayScope.initialGrayShopLimit = 2; } },
  { id: 'OD-11', name: 'gray SKU above 100 fails', check: 'initialGraySkuLimit', mutate(bundle) { bundle.owner.grayScope.initialGraySkuLimit = 101; } },
  { id: 'OD-12', name: 'pre-production optional fails', check: 'independentPreproductionRequired', mutate(bundle) { bundle.owner.environmentAndRecovery.independentPreproductionRequired = false; } },
  { id: 'OD-13', name: 'RPO above 15 fails', check: 'rpoMinutesMax', mutate(bundle) { bundle.owner.environmentAndRecovery.rpoMinutesMax = 16; } },
  { id: 'OD-14', name: 'RTO above 60 fails', check: 'rtoMinutesMax', mutate(bundle) { bundle.owner.environmentAndRecovery.rtoMinutesMax = 61; } },
  { id: 'OD-15', name: 'repository baseline silently waived fails', check: 'repositoryBaselineDisposition', mutate(bundle) { bundle.plan.repositoryBaseline.silentlyWaived = true; } },
  { id: 'OD-16', name: 'only Owner Gray approval fails', check: 'grayApprovalMode', mutate(bundle) { bundle.owner.grayApprovalRequiredApprovers = ['owner']; } },
  { id: 'OD-17', name: 'only Technical Lead Gray approval fails', check: 'grayApprovalMode', mutate(bundle) { bundle.owner.grayApprovalRequiredApprovers = ['technical_lead']; } },
  { id: 'OD-18', name: 'productionReady true fails', check: 'productionReady', mutate(bundle) { bundle.owner.productionReady = true; } },
  { id: 'OD-19', name: 'Production Acceptance true fails', check: 'productionAcceptancePassed', mutate(bundle) { bundle.owner.productionAcceptancePassed = true; } },
  { id: 'OD-20', name: 'Tag created fails', check: 'tagCreated', mutate(bundle) { bundle.gitState.tagCreated = true; } },
  { id: 'OD-21', name: 'Release created fails', check: 'releaseCreated', mutate(bundle) { bundle.owner.releaseCreated = true; } },
  { id: 'OD-22', name: 'real platform network already enabled fails', check: 'runtimeCapabilitiesDisabled', mutate(bundle) { bundle.owner.runtimeBoundary.realPlatformNetworkEnabled = true; } },
  { id: 'OD-23', name: 'secret detected fails', check: 'realSecretCount', mutate(bundle) { bundle.credentialScan = { realSecretCount: 1, credentialValueRecorded: true }; } },
];

assert.equal(fixtures.length, 23);
assert.deepEqual(fixtures.map((item) => item.id), Array.from({ length: 23 }, (_, index) => `OD-${String(index + 1).padStart(2, '0')}`));

for (const fixture of fixtures) {
  const bundle = validBundle();
  fixture.mutate(bundle);
  const report = validateP10OwnerDecisionBundle(bundle);
  if (fixture.pass) {
    assert.equal(report.status, 'passed', `${fixture.id} ${fixture.name}: ${report.failed.join(', ')}`);
    assert.equal(report.failedCount, 0, fixture.id);
  } else {
    assert.equal(report.status, 'failed', `${fixture.id} ${fixture.name}`);
    assert.ok(report.failed.includes(fixture.check), `${fixture.id} expected ${fixture.check}, got ${report.failed.join(', ')}`);
  }
}

console.log(`p10 owner decision fixtures passed: ${fixtures.length}/${fixtures.length}`);
