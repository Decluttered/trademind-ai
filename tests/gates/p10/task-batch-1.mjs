import assert from 'node:assert/strict';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  readEnvFile,
  validateP10ExternalInfrastructureEvidence,
  validateP10PreproductionContract,
} from '../../../scripts/p10-preproduction-contract.mjs';
import { validateP10TaskBatch1Bundle } from '../../../scripts/p10-task-batch-1-gate.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');

function validEnv() {
  return { ...readEnvFile(path.join(repoRoot, '.env.staging.example')) };
}

function validExternalEvidence() {
  return {
    externalInfrastructureStatus: 'provisioned',
    preproductionHostAvailable: true,
    preproductionDatabaseProvisioned: true,
    databaseIsolationProven: true,
    preproductionRedisProvisioned: true,
    redisIsolationProven: true,
    preproductionDomainProvisioned: true,
    deploymentCredentialAvailable: true,
    deploymentRehearsed: true,
    teardownRehearsed: true,
  };
}

function validBundle() {
  const externalEvidence = validExternalEvidence();
  return {
    plan: { batches: [{ batchId: 'P10-B1', taskIds: ['P10-101', 'P10-102', 'P10-103', 'P10-104'] }] },
    transition: { status: 'passed', failedCount: 0, p9ClosureReuseEligible: true, p10PlanningEntryAllowed: true },
    planningGate: { status: 'passed', failedCount: 0, planningPackCurrentHeadValid: true },
    ownerGate: { status: 'passed', failedCount: 0, p10OwnerDecisionApproved: true, p10ExecutionPlanFinalized: true },
    revalidation: {
      initialPlanningCheckpoint: 'historical-head', currentPlanningValidationHead: 'current-head',
      currentRunBaseHead: 'current-head', planningSemanticRevalidationPassed: true,
      planningSemanticManifestSha256: 'planning-manifest',
    },
    planningSemanticManifest: { sha256: 'planning-manifest' },
    frozenP9: { sha256: 'p9-protected', gitHead: 'current-head' },
    liveP9: { sha256: 'p9-protected' },
    gitState: { currentBranch: 'dev', currentHead: 'current-head', stagedFileCount: 0 },
    contract: validateP10PreproductionContract(validEnv()),
    externalEvidence,
    external: validateP10ExternalInfrastructureEvidence(externalEvidence),
    repository: {
      composeIsolation: true, readinessFoundation: true, migrationFoundation: true,
      backupFoundation: true, restoreFoundation: true, rollbackFoundation: true,
      teardownFoundation: true, unknownEnvironmentFailClosed: true, startupSafety: true,
      productionRestoreEnabled: false,
    },
    requiredFilesPresent: true,
    credentialScan: { realSecretCount: 0, credentialValueRecorded: false },
  };
}

const contractFixtures = [
  { id: 'P10B1-01', name: 'valid staging preproduction contract passes', pass: true, mutate() {} },
  { id: 'P10B1-02', name: 'unknown environment fails closed', check: 'environmentKnown', mutate(env) { env.APP_ENV = 'mystery'; } },
  { id: 'P10B1-03', name: 'missing environment fails closed', check: 'environmentKnown', mutate(env) { env.APP_ENV = ''; } },
  { id: 'P10B1-04', name: 'production environment rejected', check: 'preproductionEnvironmentMapping', mutate(env) { env.APP_ENV = 'production'; } },
  { id: 'P10B1-05', name: 'production database reuse rejected', check: 'databaseIsolation', mutate(env) { env.P10_DATABASE_ID = env.P10_PRODUCTION_DATABASE_ID; } },
  { id: 'P10B1-06', name: 'test database reuse rejected', check: 'databaseIsolation', mutate(env) { env.P10_DATABASE_ID = env.P10_TEST_DATABASE_ID; } },
  { id: 'P10B1-07', name: 'production Redis reuse rejected', check: 'redisIsolation', mutate(env) { env.P10_REDIS_ID = env.P10_PRODUCTION_REDIS_ID; } },
  { id: 'P10B1-08', name: 'inline secret value rejected', check: 'secretExternalization', mutate(env) { env.DB_PASSWORD = 'must-not-be-stored-here'; } },
  { id: 'P10B1-09', name: 'production session namespace reuse rejected', check: 'sessionIsolation', mutate(env) { env.P10_SESSION_NAMESPACE = env.P10_PRODUCTION_SESSION_NAMESPACE; } },
  { id: 'P10B1-10', name: 'production parent cookie domain overlap rejected', check: 'sessionIsolation', mutate(env) { env.AUTH_COOKIE_DOMAIN = '.preproduction.example.com'; } },
  { id: 'P10B1-11', name: 'real network capability rejected', check: 'capabilityDefaults', mutate(env) { env.INVENTORY_SYNC_NETWORK_ACCESS = 'true'; } },
  { id: 'P10B1-12', name: 'real provider mode rejected', check: 'capabilityDefaults', mutate(env) { env.INVENTORY_SYNC_PROVIDER_MODE = 'real'; } },
  { id: 'P10B1-13', name: 'production migration target rejected', check: 'migrationSafety', mutate(env) { env.P10_MIGRATION_TARGET = 'production'; } },
  { id: 'P10B1-14', name: 'production restore enablement rejected', check: 'backupRestoreSafety', mutate(env) { env.P10_PRODUCTION_RESTORE_ENABLED = 'true'; } },
  { id: 'P10B1-15', name: 'rollback without migration compatibility rejected', check: 'rollbackFoundation', mutate(env) { env.P10_ROLLBACK_MIGRATION_COMPATIBLE = 'false'; } },
  { id: 'P10B1-16', name: 'local storage in staging rejected', check: 'stagingRuntimeSafety', mutate(env) { env.STORAGE_PROVIDER = 'local'; } },
];

for (const fixture of contractFixtures) {
  const env = validEnv();
  fixture.mutate(env);
  const report = validateP10PreproductionContract(env);
  if (fixture.pass) {
    assert.equal(report.status, 'passed', `${fixture.id}: ${report.failed.join(', ')}`);
  } else {
    assert.equal(report.status, 'failed', fixture.id);
    assert.ok(report.failed.includes(fixture.check), `${fixture.id} expected ${fixture.check}`);
  }
}

const gateFixtures = [
  { id: 'P10B1-17', name: 'fully provisioned Batch 1 passes', pass: true, mutate() {} },
  { id: 'P10B1-18', name: 'missing repository file fails', check: 'repositoryFoundation', mutate(bundle) { bundle.requiredFilesPresent = false; } },
  { id: 'P10B1-19', name: 'external infrastructure absent blocks completion', check: 'externalInfrastructureProvisioned', mutate(bundle) { bundle.externalEvidence.externalInfrastructureStatus = 'not_provisioned'; bundle.external = validateP10ExternalInfrastructureEvidence(bundle.externalEvidence); } },
  { id: 'P10B1-20', name: 'P9 protected source drift fails', check: 'p9ProtectedSourceModified', mutate(bundle) { bundle.liveP9.sha256 = 'changed'; } },
  { id: 'P10B1-21', name: 'approved planning semantic drift fails', check: 'approvedPlanningSemanticsDriftDetected', mutate(bundle) { bundle.planningSemanticManifest.sha256 = 'changed'; } },
  { id: 'P10B1-22', name: 'HEAD change during Batch 1 fails', check: 'changesCommittedDuringBatch1', mutate(bundle) { bundle.gitState.currentHead = 'changed-head'; bundle.frozenP9.gitHead = 'changed-head'; } },
  { id: 'P10B1-23', name: 'staged files fail', check: 'stagedFileCount', mutate(bundle) { bundle.gitState.stagedFileCount = 1; } },
  { id: 'P10B1-24', name: 'transition entry failure blocks', check: 'entryTransition', mutate(bundle) { bundle.transition.status = 'failed'; } },
  { id: 'P10B1-25', name: 'planning entry failure blocks', check: 'entryPlanning', mutate(bundle) { bundle.planningGate.status = 'failed'; } },
  { id: 'P10B1-26', name: 'Owner entry failure blocks', check: 'entryOwnerDecision', mutate(bundle) { bundle.ownerGate.status = 'failed'; } },
  { id: 'P10B1-27', name: 'secret detection blocks', check: 'realSecretCount', mutate(bundle) { bundle.credentialScan = { realSecretCount: 1, credentialValueRecorded: true }; } },
];

for (const fixture of gateFixtures) {
  const bundle = validBundle();
  fixture.mutate(bundle);
  const report = validateP10TaskBatch1Bundle(bundle);
  if (fixture.pass) {
    assert.equal(report.status, 'passed', `${fixture.id}: ${report.failed.join(', ')}`);
    assert.equal(report.formalTaskCompleted, 4, fixture.id);
  } else {
    assert.equal(report.status, 'failed', fixture.id);
    assert.ok(report.failed.includes(fixture.check), `${fixture.id} expected ${fixture.check}, got ${report.failed.join(', ')}`);
  }
}

const fixtures = [...contractFixtures, ...gateFixtures];
assert.equal(fixtures.length, 27);
assert.deepEqual(fixtures.map((item) => item.id), Array.from({ length: 27 }, (_, index) => `P10B1-${String(index + 1).padStart(2, '0')}`));
console.log(`p10 task batch 1 fixtures passed: ${fixtures.length}/${fixtures.length}`);
