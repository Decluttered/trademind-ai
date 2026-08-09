#!/usr/bin/env node
import assert from 'node:assert/strict';
import { validateP9ProtectedFreezePreparation } from '../../scripts/ci/prepare-p9-protected-freeze.mjs';

const closureHead = 'a'.repeat(40);
const currentHead = 'b'.repeat(40);
const base = {
  closure: {
    status: 'passed',
    developmentClosureStatus: 'passed',
    p9Complete: true,
    developmentClosurePassed: true,
    currentHeadClosureVerified: true,
    currentBranch: 'dev',
    currentHead: closureHead,
    formalTaskTotal: 38,
    formalTaskCompletedCount: 38,
    formalTaskFailedCount: 0,
    acceptanceCriteriaTotal: 15,
    acceptanceCriteriaPassedCount: 15,
    acceptanceCriteriaFailedCount: 0,
    productionReady: false,
    productionAcceptancePassed: false,
    p10BoundaryPreserved: true,
  },
  closureHead,
  currentHead,
  currentBranch: 'dev',
  headDetached: false,
  stagedFileCount: 0,
  closureHistoryAvailable: true,
  closureHeadIsAncestor: true,
  validScopeManifest: true,
  protectedChangedFiles: [],
  semanticChanges: [],
  dirtyProtectedChangedFiles: [],
  productionBoundary: {
    currentAllowedLevel: 'L0',
    productionReady: false,
    productionAcceptancePassed: false,
    realPlatformNetworkCalls: 0,
    realSecretCount: 0,
    realInventoryWriteApproved: false,
    capabilities: {
      realPlatformNetworkEnabled: false,
      realCredentialsEnabled: false,
      realInventoryReadEnabled: false,
      realInventoryWriteEnabled: false,
      inventoryMutationEnabled: false,
      backgroundWorkerEnabled: false,
      automaticRetryEnabled: false,
    },
  },
};

function expectFailure(overrides, expectedCheck) {
  const result = validateP9ProtectedFreezePreparation({ ...base, ...overrides });
  assert.equal(result.status, 'blocked');
  assert.ok(result.failed.includes(expectedCheck), `expected ${expectedCheck}, got ${result.failed.join(', ')}`);
}

const valid = validateP9ProtectedFreezePreparation(base);
assert.equal(valid.status, 'passed');
assert.deepEqual(valid.failed, []);

expectFailure({ protectedChangedFiles: ['backend/internal/modules/inventorysyncp9/service.go'] }, 'protectedSourceChanges');
expectFailure({ semanticChanges: [{ id: 'p9-package-scripts', path: 'package.json' }] }, 'semanticRuleChanges');
expectFailure({ closureHistoryAvailable: false }, 'closureHistoryAvailable');
expectFailure({ closureHeadIsAncestor: false }, 'closureHeadIsAncestor');
expectFailure({ dirtyProtectedChangedFiles: ['scripts/p9-postgres-runtime.mjs'] }, 'dirtyProtectedSource');
expectFailure({ productionBoundary: { ...base.productionBoundary, currentAllowedLevel: 'L1' } }, 'productionBoundary');

console.log(JSON.stringify({
  suite: 'prepare-p9-protected-freeze',
  status: 'passed',
  fixtures: 7,
  covered: [
    'unchanged protected scope',
    'committed protected source change',
    'semantic rule change',
    'closure history unavailable',
    'closure is not ancestor',
    'dirty protected source change',
    'production boundary promotion',
  ],
}, null, 2));
