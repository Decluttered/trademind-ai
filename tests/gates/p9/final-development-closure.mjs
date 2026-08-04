import assert from 'node:assert/strict';
import { validateP9FinalDevelopmentClosureBundle } from '../../../scripts/p9-final-development-closure-gate.mjs';

const productTaskIds = [
  'P9-501', 'P9-502', 'P9-503', 'P9-504', 'P9-505', 'P9-506',
  'P9-601', 'P9-602', 'P9-603', 'P9-604', 'P9-605', 'P9-606',
  'P9-701', 'P9-702', 'P9-703', 'P9-704', 'P9-705', 'P9-706',
  'P9-801', 'P9-802', 'P9-803', 'P9-804',
  'P9-901', 'P9-902', 'P9-903', 'P9-904', 'P9-905',
  'P9-1001', 'P9-1002', 'P9-1003', 'P9-1004', 'P9-1005', 'P9-1006',
  'P9-1101', 'P9-1102', 'P9-1103', 'P9-1104', 'P9-1105',
];
const acceptanceIds = Array.from({ length: 15 }, (_, i) => `AC-P9-${String(i + 1).padStart(2, '0')}`);
const workstreams = [
  {
    workstreamId: 'WS-PRODUCT',
    workstreamType: 'product_implementation',
    tasks: productTaskIds.map((taskId, index) => ({
      taskId,
      taskCategory: 'product_implementation',
      planningFoundation: false,
      batch: index < 6 ? 1 : index < 12 ? 2 : index < 18 ? 3 : index < 22 ? 4 : index < 27 ? 5 : index < 33 ? 6 : 7,
      status: 'completed',
      acceptanceCriteriaIds: [acceptanceIds[index % acceptanceIds.length]],
    })),
  },
];
workstreams[0].tasks[0].acceptanceCriteriaIds = acceptanceIds;

const plan = {
  phaseStatus: 'Development Complete',
  executionStatus: 'development_complete',
  p9Complete: true,
  p9DevelopmentClosurePassed: true,
  productionReady: false,
  productionAcceptancePassed: false,
  productCompletedTaskCount: 38,
  batch7Completed: true,
  postgresIntegrationPassed: true,
  p10BoundaryPreserved: true,
  workstreams,
};
const closure = {
  status: 'passed',
  developmentClosureStatus: 'passed',
  p9Complete: true,
  developmentClosurePassed: true,
  acceptanceCriteriaPassedIds: acceptanceIds,
  acceptanceCriteriaPassedCount: 15,
  postgresIntegrationPassed: true,
  adminE2EPassed: true,
  adminResponsiveE2EPassed: true,
  validationCommands: [{ command: 'pnpm p9:final-development-closure-gate', status: 'passed' }],
  p10BoundaryPreserved: true,
  productionReady: false,
  productionAcceptancePassed: false,
  realDouyinProviderImplemented: false,
  oauthImplemented: false,
  realPlatformReadEnabled: false,
  realPlatformWriteEnabled: false,
  inventoryMutationEnabled: false,
  workerImplemented: false,
  backgroundSyncWorkerImplemented: false,
  automaticRetryWorkerImplemented: false,
  productionTagCreated: false,
  releaseCreated: false,
  integrationDefectFixes: [],
};
const batch7Evidence = { status: 'completed' };
const batch7Runtime = {};
const gateReports = Object.fromEntries([
  'docs/p9-plan-final-gate.json',
  'docs/p9-task-batch-1-scope-gate.json',
  'docs/p9-task-batch-1-domain-persistence-gate.json',
  'docs/p9-task-batch-2-sku-calibration-gate.json',
  'docs/p9-task-batch-3-sync-orchestration-gate.json',
  'docs/p9-task-batch-4-permissions-audit-safety-gate.json',
  'docs/p9-task-batch-5-backend-apis-gate.json',
  'docs/p9-postgresql-integration-closure-gate.json',
  'docs/p9-task-batch-6-admin-inventory-center-gate.json',
  'docs/p9-task-batch-7-integration-development-closure-gate.json',
].map((path) => [path, { status: 'passed' }]));
gateReports.batch7Validation = { status: 'passed' };
const gitState = { currentBranch: 'dev', currentHead: 'abc123', headDetached: false, stagedFileCount: 0 };

function validate(overrides = {}) {
  return validateP9FinalDevelopmentClosureBundle({
    closure: { ...closure, ...(overrides.closure || {}) },
    plan: { ...plan, ...(overrides.plan || {}) },
    batch7Evidence: { ...batch7Evidence, ...(overrides.batch7Evidence || {}) },
    batch7Runtime: { ...batch7Runtime, ...(overrides.batch7Runtime || {}) },
    gateReports: { ...gateReports, ...(overrides.gateReports || {}) },
    gitState: { ...gitState, ...(overrides.gitState || {}) },
    requiredFilesPresent: true,
  });
}

function expectFailed(check, overrides = {}) {
  const result = validate(overrides);
  assert.equal(result.status, 'failed');
  assert.ok(result.failed.includes(check), `${check} should fail; actual failures: ${result.failed.join(', ')}`);
}

assert.equal(validate().status, 'passed');
expectFailed('closureStatus', { closure: { p9Complete: false } });
expectFailed('planStatus', { plan: { phaseStatus: 'In Progress' } });
expectFailed('productTaskIdsPreserved', { plan: { workstreams: [{ ...workstreams[0], tasks: workstreams[0].tasks.slice(1) }] } });
expectFailed('productTasksCompleted', { plan: { productCompletedTaskCount: 37, workstreams: [{ ...workstreams[0], tasks: workstreams[0].tasks.map((task, index) => index === 0 ? { ...task, status: 'planned' } : task) }] } });
expectFailed('batch7Completed', { plan: { batch7Completed: false } });
expectFailed('acceptanceCriteriaPassed', { closure: { acceptanceCriteriaPassedIds: acceptanceIds.slice(0, 14), acceptanceCriteriaPassedCount: 14 } });
expectFailed('historicalGatesPassed', { gateReports: { ...gateReports, 'docs/p9-task-batch-4-permissions-audit-safety-gate.json': { status: 'failed' } } });
expectFailed('postgresIntegrationPassed', { closure: { postgresIntegrationPassed: false } });
expectFailed('adminE2EPassed', { closure: { adminE2EPassed: false } });
expectFailed('qualityGateEvidenceRecorded', { closure: { validationCommands: [] } });
expectFailed('p10BoundaryPreserved', { closure: { p10BoundaryPreserved: false } });
expectFailed('productionBoundary', { closure: { productionReady: true } });
expectFailed('productionBoundary', { closure: { productionAcceptancePassed: true } });
expectFailed('productionBoundary', { closure: { realDouyinProviderImplemented: true } });
expectFailed('integrationDefectFixes', { closure: { integrationDefectFixes: [{ id: 'fix' }] } });
expectFailed('stagedFileCount', { gitState: { stagedFileCount: 1 } });

console.log('p9 final development closure fixtures passed');
