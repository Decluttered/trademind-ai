import assert from 'node:assert/strict';
import { validateP10PlanningPackBundle } from '../../../scripts/p10-planning-pack-gate.mjs';

function decision(index) {
  return {
    decisionId: `D${String(index).padStart(2, '0')}`,
    question: `Decision ${index}`,
    options: ['recommended', 'alternative'],
    recommendedOption: 'recommended',
    recommendationReason: 'bounded recommendation',
    risk: 'owner decision required',
    dependencies: [],
    ownerApprovalRequired: true,
    recommendationOnly: true,
    approved: false,
    status: 'pending_owner_approval',
  };
}

function validBundle() {
  return {
    transition: {
      status: 'passed',
      failedCount: 0,
      p9ClosureReuseEligible: true,
      p10PlanningEntryAllowed: true,
      p9ProtectedChangedFileCount: 0,
      dirtyProtectedSourceDriftDetected: false,
    },
    freeze: { sha256: 'frozen-source', gitHead: 'base-head' },
    liveProtectedSourceManifest: { sha256: 'frozen-source' },
    proposal: {
      ownerDecisionCount: 15,
      ownerApprovalPending: true,
      recommendationOnly: true,
      approved: false,
      decisions: Array.from({ length: 15 }, (_, index) => decision(index + 1)),
    },
    boundary: {
      currentAllowedLevel: 'L0',
      levels: [{}, {}, {}, {}],
      approvals: {
        realProviderApproved: false,
        realInventoryReadApproved: false,
        realInventoryWriteApproved: false,
        backgroundWorkerApproved: false,
        automaticRetryApproved: false,
      },
      runtimeBoundary: {
        realPlatformNetworkEnabled: false,
        realCredentialsEnabled: false,
        inventoryMutationEnabled: false,
        backgroundWorkerEnabled: false,
        automaticRetryEnabled: false,
      },
      productionReady: false,
      productionAcceptancePassed: false,
    },
    plan: {
      workstreams: [
        ...Array.from({ length: 10 }, (_, index) => ({ workstreamId: `P10-W${index + 1}` })),
        { workstreamId: 'P10-W11', conditional: true, status: 'blocked_pending_owner_approval' },
      ],
      rollbackPlan: {
        postgresqlBackup: true,
        pointInTimeRecovery: true,
        restoreDrill: true,
        deploymentRollback: true,
        migrationRecovery: true,
        credentialRevocation: true,
      },
      killSwitches: ['provider', 'tenant', 'shop', 'read', 'write'],
      performanceClosurePlan: {
        status: 'open_blocking_for_production',
        historicalFacts: { completedRunCount: 2, expectedRunCount: 4, failedCount: 8, validForFormalPlan: false },
        requiredEvidence: ['P50, P95, and P99 latency', 'Throughput and concurrency', 'CPU and memory', 'Database connections', 'Provider rate-limit behavior'],
        historicalWaiverIsProductionAcceptance: false,
      },
      repositoryBaseline: {
        disposition: 'pending_owner_approval',
        silentlyWaived: false,
        knownItems: ['architecture loader', 'Go formatting', 'affected cascade'],
        options: ['A', 'B', 'C'],
      },
    },
    criteria: { acceptanceCriteria: Array.from({ length: 24 }, () => ({})), productionAcceptancePassed: false },
    risks: { risks: Array.from({ length: 16 }, () => ({})) },
    pack: {
      planningStatus: 'prepared_for_owner_review',
      p10PlanningPackPrepared: true,
      p10ImplementationStarted: false,
      p9ProtectedSourceModified: false,
      p9ProtectedSourceModifiedDuringPlanning: false,
      p9ProtectedSourceDriftDetected: false,
      ownerDecisionProposalPresent: true,
      ownerApprovalPending: true,
      unapprovedDecisionTreatedAsApproved: false,
      productionBoundaryPresent: true,
      executionPlanDraftPresent: true,
      acceptanceCriteriaDraftPresent: true,
      riskRegisterPresent: true,
      realProviderApproved: false,
      realInventoryReadApproved: false,
      realInventoryWriteApproved: false,
      backgroundWorkerApproved: false,
      automaticRetryApproved: false,
      repositoryBaselineDisposition: 'pending_owner_approval',
      productionReady: false,
      productionAcceptancePassed: false,
      tagCreated: false,
      releaseCreated: false,
      credentialValueRecorded: false,
      baseBranch: 'dev',
      baseCheckpoint: 'base-head',
      changesCommitted: false,
      stagedFileCount: 0,
    },
    gitState: { currentBranch: 'dev', currentHead: 'base-head', stagedFileCount: 0, tagCreated: false },
    requiredFilesPresent: true,
    credentialScan: { realSecretCount: 0, credentialValueRecorded: false },
  };
}

const fixtures = [
  { id: 'P10P-01', name: 'valid pack passes', expected: 'passed', mutate() {} },
  { id: 'P10P-02', name: 'transition gate blocked fails', expectedCheck: 'p9ToP10TransitionPassed', mutate(bundle) { bundle.transition.status = 'blocked'; } },
  { id: 'P10P-03', name: 'P9 protected source changed fails', expectedCheck: 'p9ProtectedSourceModified', mutate(bundle) { bundle.liveProtectedSourceManifest.sha256 = 'changed'; } },
  { id: 'P10P-04', name: 'missing proposal fails', expectedCheck: 'ownerDecisionProposalPresent', mutate(bundle) { bundle.pack.ownerDecisionProposalPresent = false; } },
  { id: 'P10P-05', name: 'decision count not 15 fails', expectedCheck: 'ownerDecisionCount', mutate(bundle) { bundle.proposal.decisions.pop(); bundle.proposal.ownerDecisionCount = 14; } },
  { id: 'P10P-06', name: 'recommendation treated as approval fails', expectedCheck: 'ownerDecisionCount', mutate(bundle) { bundle.proposal.decisions[0].approved = true; } },
  { id: 'P10P-07', name: 'real provider approval without evidence fails', expectedCheck: 'realProviderApproved', mutate(bundle) { bundle.pack.realProviderApproved = true; } },
  { id: 'P10P-08', name: 'real read approval without evidence fails', expectedCheck: 'realInventoryReadApproved', mutate(bundle) { bundle.pack.realInventoryReadApproved = true; } },
  { id: 'P10P-09', name: 'real write approval without evidence fails', expectedCheck: 'realInventoryWriteApproved', mutate(bundle) { bundle.pack.realInventoryWriteApproved = true; } },
  { id: 'P10P-10', name: 'worker approval without evidence fails', expectedCheck: 'backgroundWorkerApproved', mutate(bundle) { bundle.pack.backgroundWorkerApproved = true; } },
  { id: 'P10P-11', name: 'retry approval without evidence fails', expectedCheck: 'automaticRetryApproved', mutate(bundle) { bundle.pack.automaticRetryApproved = true; } },
  { id: 'P10P-12', name: 'missing production boundary fails', expectedCheck: 'productionBoundaryPresent', mutate(bundle) { bundle.pack.productionBoundaryPresent = false; } },
  { id: 'P10P-13', name: 'missing rollback fails', expectedCheck: 'rollbackPlanPresent', mutate(bundle) { bundle.plan.rollbackPlan.restoreDrill = false; } },
  { id: 'P10P-14', name: 'missing kill switch fails', expectedCheck: 'killSwitchPlanPresent', mutate(bundle) { bundle.plan.killSwitches = ['provider', 'tenant', 'shop', 'read']; } },
  { id: 'P10P-15', name: 'missing P7 performance closure plan fails', expectedCheck: 'p7PerformanceClosurePlanPresent', mutate(bundle) { bundle.plan.performanceClosurePlan.requiredEvidence = []; } },
  { id: 'P10P-16', name: 'repository baseline silently waived fails', expectedCheck: 'repositoryBaselineDispositionPending', mutate(bundle) { bundle.plan.repositoryBaseline.silentlyWaived = true; } },
  { id: 'P10P-17', name: 'productionReady true fails', expectedCheck: 'productionReady', mutate(bundle) { bundle.pack.productionReady = true; } },
  { id: 'P10P-18', name: 'productionAcceptancePassed true fails', expectedCheck: 'productionAcceptancePassed', mutate(bundle) { bundle.pack.productionAcceptancePassed = true; } },
  { id: 'P10P-19', name: 'tag created fails', expectedCheck: 'tagCreated', mutate(bundle) { bundle.gitState.tagCreated = true; } },
  { id: 'P10P-20', name: 'release created fails', expectedCheck: 'releaseCreated', mutate(bundle) { bundle.pack.releaseCreated = true; } },
  { id: 'P10P-21', name: 'secret detected fails', expectedCheck: 'realSecretCount', mutate(bundle) { bundle.credentialScan = { realSecretCount: 1, credentialValueRecorded: true }; } },
  { id: 'P10P-22', name: 'staged files fails', expectedCheck: 'stagedFileCount', mutate(bundle) { bundle.gitState.stagedFileCount = 1; } },
];

for (const fixture of fixtures) {
  const bundle = validBundle();
  fixture.mutate(bundle);
  const report = validateP10PlanningPackBundle(bundle);
  if (fixture.expected === 'passed') {
    assert.equal(report.status, 'passed', `${fixture.id} ${fixture.name}`);
    assert.equal(report.failedCount, 0, `${fixture.id} ${fixture.name}`);
  } else {
    assert.equal(report.status, 'failed', `${fixture.id} ${fixture.name}`);
    assert.ok(report.failed.includes(fixture.expectedCheck), `${fixture.id} expected ${fixture.expectedCheck}, got ${report.failed.join(', ')}`);
  }
}

console.log(`p10 planning pack fixtures passed: ${fixtures.length}/${fixtures.length}`);
