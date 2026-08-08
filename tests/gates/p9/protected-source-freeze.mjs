import assert from 'node:assert/strict';
import {
  hashProtectedSourceEntries,
  validateProtectedSourceFreezeBundle,
} from '../../../scripts/p9-protected-source-freeze.mjs';

const entries = [
  { path: 'backend/p9.go', category: 'product_source', categories: ['product_source'], hashKind: 'file', sha256: 'a'.repeat(64) },
  { path: 'package.json', category: 'runtime_contract', categories: ['runtime_contract'], hashKind: 'semantic:package_script_prefixes', sha256: 'b'.repeat(64) },
];
const manifestSha256 = hashProtectedSourceEntries(entries);
const live = {
  gitHead: 'abc123',
  currentBranch: 'dev',
  sha256: manifestSha256,
  fileCount: entries.length,
  dirtyProtectedChangedFiles: ['backend/p9.go'],
  entries,
};
const freeze = {
  manifestType: 'p9_protected_source_freeze',
  gitHead: 'abc123',
  currentBranch: 'dev',
  sha256: manifestSha256,
  fileCount: entries.length,
  entries,
};

assert.equal(validateProtectedSourceFreezeBundle({ freeze, live }).status, 'passed');
assert.equal(hashProtectedSourceEntries([...entries].reverse()), manifestSha256);

const drifted = { ...live, sha256: 'c'.repeat(64) };
const driftResult = validateProtectedSourceFreezeBundle({ freeze, live: drifted });
assert.equal(driftResult.status, 'failed');
assert.equal(driftResult.protectedSourceDriftDetected, true);
assert.ok(driftResult.failed.includes('liveManifestHash'));

const wrongHead = validateProtectedSourceFreezeBundle({ freeze, live: { ...live, gitHead: 'def456' } });
assert.equal(wrongHead.status, 'failed');
assert.ok(wrongHead.failed.includes('gitHead'));

const incomplete = validateProtectedSourceFreezeBundle({
  freeze: { ...freeze, entries: entries.slice(1), fileCount: 1, sha256: hashProtectedSourceEntries(entries.slice(1)) },
  live,
});
assert.equal(incomplete.status, 'failed');
assert.ok(incomplete.failed.includes('liveManifestHash'));
assert.ok(incomplete.failed.includes('dirtyProtectedFilesIncluded'));

console.log('p9 protected source freeze fixtures passed');
