import assert from 'node:assert/strict';
import {
  hashSourceManifestEntries,
  validateP9ToP10TransitionBundle,
} from '../../../scripts/p9-to-p10-transition-gate.mjs';

const closureHead = 'a'.repeat(40);
const descendantHead = 'b'.repeat(40);
const taskIds = Array.from({ length: 38 }, (_, index) => `P9-${500 + index}`);
const acceptanceIds = Array.from({ length: 15 }, (_, index) => `AC-P9-${String(index + 1).padStart(2, '0')}`);
const tasks = taskIds.map((taskId, index) => ({
  taskId,
  taskCategory: 'product_implementation',
  planningFoundation: false,
  status: 'completed',
  acceptanceCriteriaIds: index === 0 ? acceptanceIds : [acceptanceIds[index % acceptanceIds.length]],
}));
const plan = {
  phaseStatus: 'Development Complete',
  executionStatus: 'development_complete',
  p9Complete: true,
  p9DevelopmentClosurePassed: true,
  productionReady: false,
  productionAcceptancePassed: false,
  workstreams: [{ workstreamId: 'P9-W', tasks }],
};
const closure = {
  status: 'passed',
  developmentClosureStatus: 'passed',
  p9Complete: true,
  developmentClosurePassed: true,
  currentBranch: 'dev',
  currentHead: closureHead,
  currentClosureHead: closureHead,
  currentHeadClosureVerified: true,
  formalTaskTotal: 38,
  formalTaskCompletedCount: 38,
  formalTaskFailedCount: 0,
  formalTaskDeferredCount: 0,
  acceptanceCriteriaTotal: 15,
  acceptanceCriteriaPassedCount: 15,
  acceptanceCriteriaFailedCount: 0,
  postgresRuntimeRunId: 'pg-run',
  postgresRuntimeHead: closureHead,
  postgresRuntimeSummarySha256: 'pg-sha',
  postgresRuntimeProtectedSourceManifestSha256: '',
  batch7RuntimeRunId: 'b7-run',
  batch7RuntimeHead: closureHead,
  batch7RuntimeSummarySha256: 'b7-sha',
  batch7RuntimeProtectedSourceManifestSha256: '',
  protectedSourceManifestSha256: '',
  productionReady: false,
  productionAcceptancePassed: false,
};
const postgresRuntime = {
  runId: 'pg-run',
  status: 'passed',
  git: { endHead: closureHead, stable: true },
};
const sourceEntries = [{ path: 'backend/internal/modules/inventorysyncp9/model.go', sha256: 'source-sha' }];
const sourceManifestSha = hashSourceManifestEntries(sourceEntries);
closure.protectedSourceManifestSha256 = sourceManifestSha;
closure.postgresRuntimeProtectedSourceManifestSha256 = sourceManifestSha;
closure.batch7RuntimeProtectedSourceManifestSha256 = sourceManifestSha;
postgresRuntime.protectedSourceFreeze = { sha256: sourceManifestSha };
const sourceManifest = { sha256: sourceManifestSha, gitHead: closureHead, entries: sourceEntries };
const batch7Runtime = {
  runId: 'b7-run',
  status: 'passed',
  completed: true,
  currentHead: closureHead,
  sourceManifestSha256: sourceManifestSha,
  sourceManifestHead: closureHead,
  protectedSourceManifestSha256: sourceManifestSha,
  productionReady: false,
  productionAcceptancePassed: false,
};
const batch7Evidence = {
  status: 'completed',
  integrationStatus: 'passed',
  currentHead: closureHead,
  runtimeEvidence: { sourceManifestSha256: sourceManifestSha, sourceManifestHead: closureHead, protectedSourceManifestSha256: sourceManifestSha },
};
const scopeManifest = {
  schemaVersion: 1,
  manifestType: 'p9_protected_scope',
  blockingCategories: {
    P9_PRODUCT_SOURCE: { paths: ['backend/internal/modules/inventorysyncp9/**'] },
    P9_SECURITY_AND_PERSISTENCE_CONTRACT: { paths: ['backend/internal/testing/postgrestest/**', 'backend/internal/modules/product/sku_search.go'] },
    P9_RUNTIME_CONTRACT: { paths: ['scripts/p9-postgres-runtime.mjs'] },
    P9_GATE_SEMANTICS: { paths: ['scripts/p9-final-development-closure-gate.mjs', 'tests/gates/p9/**'] },
    P9_ADMIN_PRODUCT_SOURCE: { paths: ['admin/src/pages/Ops/InventorySync/**'] },
    P9_API_CONTRACT: { paths: ['docs/api.md'] },
    P9_SCOPE_AND_ACCEPTANCE: { paths: ['docs/p9-execution-plan.json'] },
  },
  nonBlockingGeneratedEvidence: { paths: ['docs/P9_*_GATE.md', 'docs/p9-*-gate.json'] },
};
const gitState = {
  currentBranch: 'dev',
  currentHead: descendantHead,
  headDetached: false,
  stagedFileCount: 0,
  p9ClosureHeadIsAncestor: true,
};

function validate(overrides = {}) {
  const effectiveGitState = { ...gitState, ...(overrides.gitState || {}) };
  return validateP9ToP10TransitionBundle({
    closure: { ...closure, ...(overrides.closure || {}) },
    plan: { ...plan, ...(overrides.plan || {}) },
    postgresRuntime: { ...postgresRuntime, ...(overrides.postgresRuntime || {}) },
    batch7Runtime: { ...batch7Runtime, ...(overrides.batch7Runtime || {}) },
    batch7Evidence: { ...batch7Evidence, ...(overrides.batch7Evidence || {}) },
    sourceManifest: overrides.sourceManifest || sourceManifest,
    liveProtectedSourceManifest: overrides.liveProtectedSourceManifest || { sha256: sourceManifestSha, gitHead: effectiveGitState.currentHead, entries: sourceEntries },
    scopeManifest: overrides.scopeManifest || scopeManifest,
    gitState: effectiveGitState,
    changedFiles: overrides.changedFiles || [],
    semanticChanges: overrides.semanticChanges || [],
    artifactHashes: {
      postgresRuntimeSha256: 'pg-sha',
      batch7RuntimeSha256: 'b7-sha',
      protectedScopeManifestSha256: 'scope-sha',
      ...(overrides.artifactHashes || {}),
    },
    requiredFilesPresent: overrides.requiredFilesPresent ?? true,
    commitsSinceP9Closure: [],
  });
}

function expectPassed(id, overrides = {}) {
  const report = validate(overrides);
  assert.equal(report.status, 'passed', `${id} should pass; failures: ${report.failed.join(', ')}`);
  assert.equal(report.p9ClosureReuseEligible, true);
  assert.equal(report.p10PlanningEntryAllowed, true);
}
function expectBlocked(id, failedCheck, overrides = {}) {
  const report = validate(overrides);
  assert.equal(report.status, 'blocked', `${id} should be blocked`);
  assert.ok(report.failed.includes(failedCheck), `${id} should fail ${failedCheck}; failures: ${report.failed.join(', ')}`);
  assert.equal(report.p10PlanningEntryAllowed, false);
}

expectPassed('TR-01 historical closure descendant without P9 changes');
expectPassed('TR-02 current HEAD equals closure HEAD', { gitState: { currentHead: closureHead } });
expectBlocked('TR-03 closure HEAD not ancestor', 'p9ClosureHeadIsAncestor', { gitState: { p9ClosureHeadIsAncestor: false } });
expectBlocked('TR-04 inventory source changed', 'p9ProductSourceChanged', { changedFiles: ['backend/internal/modules/inventorysyncp9/model.go'] });
expectBlocked('TR-05 Admin source changed', 'p9AdminProductSourceChanged', { changedFiles: ['admin/src/pages/Ops/InventorySync/index.tsx'] });
expectBlocked('TR-06 API contract changed', 'p9ApiContractChanged', { changedFiles: ['docs/api.md'] });
expectBlocked('TR-07 runtime script changed', 'p9RuntimeContractChanged', { changedFiles: ['scripts/p9-postgres-runtime.mjs'] });
expectBlocked('TR-08 gate logic changed', 'p9GateSemanticsChanged', { changedFiles: ['scripts/p9-final-development-closure-gate.mjs'] });
expectBlocked('TR-09 tenant isolation changed', 'p9SecurityContractChanged', { changedFiles: ['backend/internal/modules/product/sku_search.go'] });
expectBlocked('TR-10 acceptance criteria changed', 'p9ScopeChanged', { changedFiles: ['docs/p9-execution-plan.json'] });
expectPassed('TR-11 only P10 planning docs changed', { changedFiles: ['docs/P10_OWNER_DECISION_PROPOSAL.md'] });
expectPassed('TR-12 only unrelated module changed', { changedFiles: ['backend/internal/modules/orders/service.go'] });
expectPassed('TR-13 generated report timestamp changed', { changedFiles: ['docs/p9-final-development-closure-gate.json'] });
expectBlocked('TR-14 invalid closure SHA-256', 'postgresArtifactSha256Valid', { artifactHashes: { postgresRuntimeSha256: 'tampered' } });
expectBlocked('TR-15 PostgreSQL runtime HEAD mismatch', 'postgresRuntimeHeadMatchesClosureHead', { postgresRuntime: { git: { endHead: descendantHead, stable: true } } });
expectBlocked('TR-16 Batch 7 runtime HEAD mismatch', 'batch7RuntimeHeadMatchesClosureHead', { batch7Runtime: { currentHead: descendantHead } });
expectBlocked('TR-17 p9Complete false', 'p9Complete', { closure: { p9Complete: false } });
expectBlocked('TR-18 productionReady true', 'productionReady', { closure: { productionReady: true } });
expectBlocked('TR-19 same HEAD but dirty P9 protected source changed', 'dirtyProtectedSourceDriftDetected', { gitState: { currentHead: closureHead }, liveProtectedSourceManifest: { sha256: 'd'.repeat(64), gitHead: closureHead, entries: sourceEntries } });
expectPassed('TR-20 same HEAD but only generated evidence changed', { gitState: { currentHead: closureHead }, changedFiles: ['docs/p9-final-development-closure-gate.json'] });

console.log('p9-to-p10 transition fixtures passed: 20/20');
