import assert from 'node:assert/strict';
import { validateP9Batch7IntegrationBundle } from '../../../scripts/p9-task-batch-7-e2e-gate.mjs';
import { hashProtectedSourceEntries } from '../../../scripts/p9-protected-source-freeze.mjs';

const protectedSourceSha = hashProtectedSourceEntries([]);

const tasks = Object.fromEntries(['P9-1101', 'P9-1102', 'P9-1103', 'P9-1104', 'P9-1105'].map((id) => [id, { status: 'completed' }]));
const evidence = {
  batchId: 'P9-TASK-BATCH-7',
  currentBranch: 'dev',
  currentHead: 'abc123',
  status: 'completed',
  integrationStatus: 'passed',
  formalTaskTotal: 5,
  formalTaskCompletedCount: 5,
  tasks,
  acceptanceCriteriaPassedIds: ['AC-P9-09', 'AC-P9-10', 'AC-P9-11', 'AC-P9-12', 'AC-P9-13', 'AC-P9-14', 'AC-P9-15'],
  runtimeEvidence: { runId: 'p9b7-run', runtimeSummarySha256: 'runtime-sha', sourceManifestSha256: 'manifest-sha', protectedSourceManifestSha256: protectedSourceSha, runtimeJsonlSha256: 'runtime-jsonl-sha', e2eArtifactSha256: 'e2e-sha', raceArtifactSha256: 'race-sha', finishedAt: '2026-08-08T00:00:00.000Z' },
  postgresRuntimeEvidence: { runId: 'pg-run', protectedSourceManifestSha256: protectedSourceSha },
  integrationDefectFixes: [],
  p10BoundaryPreserved: true,
  productionReady: false,
  productionAcceptancePassed: false,
  realDouyinProviderImplemented: false,
  oauthImplemented: false,
  realPlatformReadEnabled: false,
  realPlatformWriteEnabled: false,
  workerImplemented: false,
  backgroundSyncWorkerImplemented: false,
  automaticRetryWorkerImplemented: false,
};
const runtime = {
  batchId: 'P9-TASK-BATCH-7',
  runId: 'p9b7-run',
  status: 'passed',
  completed: true,
  currentBranch: 'dev',
  currentHead: 'abc123',
  finishedAt: '2026-08-08T00:00:00.000Z',
  sourceManifestSha256: 'manifest-sha',
  protectedSourceManifestSha256: protectedSourceSha,
  protectedSourceFreezeHead: 'abc123',
  protectedSourceManifestBeforeSha256: protectedSourceSha,
  protectedSourceManifestAfterSha256: protectedSourceSha,
  protectedSourceFrozen: true,
  protectedSourceStable: true,
  protectedSourceDriftDetected: false,
  runtimeJsonlSha256: 'runtime-jsonl-sha',
  e2eArtifactSha256: 'e2e-sha',
  raceArtifactSha256: 'race-sha',
  authenticatedPostgresE2E: true,
  tenantIsolationPassed: true,
  rbacMatrixPassed: true,
  crossTenantWriteDenied: true,
  idempotencyPassed: true,
  revisionConflictPassed: true,
  keysetPaginationPassed: true,
  adminE2EPassed: true,
  integrationFixtures: { success: true, lowConfidence: true, conflict: true, manualBinding: true, failure: true },
  platformBoundary: { realPlatformNetworkCalls: 0, realCredentialsUsed: false, inventoryMutationCalls: 0 },
  p10BoundaryPreserved: true,
  productionReady: false,
  productionAcceptancePassed: false,
};
const postgresRuntime = {
  runId: 'pg-run',
  git: { endHead: 'abc123' },
  protectedSourceFreeze: { sha256: protectedSourceSha },
  contracts: { postgresIntegrationPassed: true, postgresFixtureGoldenPathPassed: true },
};
const postgresGate = { status: 'passed', currentHead: 'abc123' };
const batch6Gate = { status: 'passed' };
const sources = {
  e2e: '@p9-batch7 viewports writeGuard.allow',
  mocks: 'success_single_page',
  service: 'nextCursor',
  backendIntegration: 'TestP9PGBearerAuthAndFixtureGoldenPath',
  packageJSON: 'test:p9-task-batch-7-e2e test:p9-task-batch-7-runtime p9:task-batch-7-e2e-gate',
};
const gitState = { currentBranch: 'dev', currentHead: 'abc123', headDetached: false, stagedFileCount: 0 };
const validFiles = true;

function validate(overrides = {}) {
  return validateP9Batch7IntegrationBundle({
    evidence: { ...evidence, ...(overrides.evidence || {}) },
    runtime: { ...runtime, ...(overrides.runtime || {}) },
    postgresRuntime: { ...postgresRuntime, ...(overrides.postgresRuntime || {}) },
    postgresGate: overrides.postgresGate || postgresGate,
    batch6Gate: overrides.batch6Gate || batch6Gate,
    sources: { ...sources, ...(overrides.sources || {}) },
    gitState: { ...gitState, ...(overrides.gitState || {}) },
    requiredFilesPresent: validFiles,
    sourceManifest: overrides.sourceManifest || { sha256: 'manifest-sha', currentBranch: 'dev', currentHead: 'abc123' },
    liveSourceManifest: overrides.liveSourceManifest || { sha256: 'manifest-sha' },
    protectedSourceFreeze: overrides.protectedSourceFreeze || { manifestType: 'p9_protected_source_freeze', currentBranch: 'dev', gitHead: 'abc123', sha256: protectedSourceSha, fileCount: 0, entries: [] },
    liveProtectedSourceManifest: overrides.liveProtectedSourceManifest || { currentBranch: 'dev', gitHead: 'abc123', sha256: protectedSourceSha, fileCount: 0, dirtyProtectedChangedFiles: [], entries: [] },
    runtimeArtifactSha: overrides.runtimeArtifactSha ?? 'runtime-sha',
    runtimeJsonlSha: overrides.runtimeJsonlSha ?? 'runtime-jsonl-sha',
    e2eArtifactSha: overrides.e2eArtifactSha ?? 'e2e-sha',
    raceArtifactSha: overrides.raceArtifactSha ?? 'race-sha',
  });
}

function expectFailed(check, overrides = {}) {
  const result = validate(overrides);
  assert.equal(result.status, 'failed');
  assert.ok(result.failed.includes(check), `${check} should fail; actual failures: ${result.failed.join(', ')}`);
}

assert.equal(validate().status, 'passed');
expectFailed('P9-1102 status', { evidence: { tasks: { ...tasks, 'P9-1102': { status: 'planned' } } } });
expectFailed('acceptanceCoverage', { evidence: { acceptanceCriteriaPassedIds: ['AC-P9-09'] } });
expectFailed('postgresContracts', { postgresRuntime: { contracts: { postgresIntegrationPassed: false, postgresFixtureGoldenPathPassed: true } } });
expectFailed('authenticatedE2E', { runtime: { authenticatedPostgresE2E: false } });
expectFailed('runtimeHeadBinding', { runtime: { currentHead: 'stale-head' } });
expectFailed('sourceManifestBinding', { liveSourceManifest: { sha256: 'changed-manifest' } });
expectFailed('protectedSourceManifestBinding', { runtime: { protectedSourceManifestSha256: 'changed-manifest' } });
expectFailed('protectedSourceManifestBinding', { liveProtectedSourceManifest: { currentBranch: 'dev', gitHead: 'abc123', sha256: 'e'.repeat(64), fileCount: 0, dirtyProtectedChangedFiles: [], entries: [] } });
expectFailed('protectedSourceManifestBinding', { runtime: { protectedSourceDriftDetected: true, protectedSourceStable: false } });
expectFailed('artifactHashBinding', { e2eArtifactSha: 'changed-e2e' });
expectFailed('postgresRuntimeBinding', { postgresRuntime: { git: { endHead: 'stale-head' } } });
expectFailed('tenantRbacIsolation', { runtime: { rbacMatrixPassed: false } });
expectFailed('idempotencyAndConflict', { runtime: { revisionConflictPassed: false } });
expectFailed('keysetPagination', { sources: { service: 'nextCursor offset' } });
expectFailed('adminE2E', { sources: { e2e: '@p9-inventory-sync viewports writeGuard.allow' } });
expectFailed('integrationFixtures', { runtime: { integrationFixtures: { ...runtime.integrationFixtures, conflict: false } } });
expectFailed('platformBoundaryFinalGate', { runtime: { platformBoundary: { realPlatformNetworkCalls: 1, realCredentialsUsed: false, inventoryMutationCalls: 0 } } });
expectFailed('noProductCapabilityAdded', { evidence: { integrationDefectFixes: [{ id: 'fix' }] } });
expectFailed('productionNotReady', { evidence: { productionReady: true }, runtime: { productionReady: true } });
expectFailed('noRealPlatformCapability', { evidence: { realPlatformWriteEnabled: true } });
expectFailed('stagedFileCount', { gitState: { stagedFileCount: 1 } });

console.log('p9 task batch 7 e2e fixtures passed');
