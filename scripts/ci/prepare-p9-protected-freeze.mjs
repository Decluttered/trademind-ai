#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { classifyP9ProtectedChanges } from '../p9-to-p10-transition-gate.mjs';
import {
  P9_PROTECTED_SCOPE_MANIFEST_JSON,
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  computeLiveProtectedSourceManifest,
  repoRoot,
  validateProtectedSourceFreezeBundle,
  writeProtectedSourceFreeze,
} from '../p9-protected-source-freeze.mjs';

const CLOSURE_JSON = 'docs/p9-final-development-closure.json';
const P10_DEVELOPMENT_COMPLETION_JSON = 'docs/p10-development-completion.json';
const REQUIRED_SCOPE_CATEGORIES = [
  'P9_PRODUCT_SOURCE',
  'P9_SECURITY_AND_PERSISTENCE_CONTRACT',
  'P9_RUNTIME_CONTRACT',
  'P9_GATE_SEMANTICS',
  'P9_ADMIN_PRODUCT_SOURCE',
  'P9_API_CONTRACT',
  'P9_SCOPE_AND_ACCEPTANCE',
];
const DISABLED_CAPABILITIES = [
  'realPlatformNetworkEnabled',
  'realCredentialsEnabled',
  'realInventoryReadEnabled',
  'realInventoryWriteEnabled',
  'inventoryMutationEnabled',
  'backgroundWorkerEnabled',
  'automaticRetryEnabled',
];

function rootPath(relativePath) {
  return path.join(repoRoot, relativePath);
}

function readJSON(relativePath) {
  try {
    return JSON.parse(fs.readFileSync(rootPath(relativePath), 'utf8'));
  } catch {
    return null;
  }
}

function git(args) {
  return execFileSync('git', args, { cwd: repoRoot, encoding: 'utf8' }).trim();
}

function gitExitZero(args) {
  return spawnSync('git', args, { cwd: repoRoot, encoding: 'utf8' }).status === 0;
}

function ensureClosureHistory(closureHead, currentHead) {
  if (!/^[a-f0-9]{40}$/.test(closureHead) || !/^[a-f0-9]{40}$/.test(currentHead)) return false;
  if (gitExitZero(['merge-base', '--is-ancestor', closureHead, currentHead])) return true;

  const shallow = git(['rev-parse', '--is-shallow-repository']) === 'true';
  const fetchArgs = shallow
    ? ['fetch', '--no-tags', '--unshallow', 'origin']
    : ['fetch', '--no-tags', 'origin', closureHead];
  const fetched = spawnSync('git', fetchArgs, {
    cwd: repoRoot,
    encoding: 'utf8',
    env: { ...process.env, GIT_TERMINAL_PROMPT: '0' },
    stdio: 'pipe',
  });
  return fetched.status === 0 && gitExitZero(['merge-base', '--is-ancestor', closureHead, currentHead]);
}

function normalizePath(value) {
  return String(value || '').replaceAll('\\', '/').replace(/^\.\//, '');
}

function parseNameStatus(text) {
  return String(text || '').split(/\r?\n/).filter(Boolean).map((line) => {
    const [status, ...paths] = line.split('\t');
    return { status, path: normalizePath(paths.at(-1) || '') };
  }).filter((row) => row.path);
}

function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
}

function sameValue(left, right) {
  return JSON.stringify(stable(left)) === JSON.stringify(stable(right));
}

function semanticValue(document, rule) {
  if (!document) return null;
  if (rule.kind === 'package_script_prefixes') {
    const excluded = new Set(rule.excludeKeys || []);
    return Object.fromEntries(Object.entries(document.scripts || {})
      .filter(([key]) => (rule.includePrefixes || []).some((prefix) => key.startsWith(prefix)) && !excluded.has(key))
      .sort(([left], [right]) => left.localeCompare(right)));
  }
  if (rule.kind === 'json_fields') {
    return Object.fromEntries((rule.fields || []).map((field) => [field, document[field]]));
  }
  return null;
}

function readJSONAt(ref, relativePath) {
  try {
    return JSON.parse(git(['show', `${ref}:${relativePath}`]));
  } catch {
    return null;
  }
}

function detectSemanticChanges(baseRef, scopeManifest) {
  const changes = [];
  for (const rule of scopeManifest.semanticRules || []) {
    const before = semanticValue(readJSONAt(baseRef, rule.path), rule);
    const after = semanticValue(readJSON(rule.path), rule);
    if (!sameValue(before, after)) {
      changes.push({ id: rule.id, path: rule.path, category: rule.category });
    }
  }
  return changes;
}

function scopeManifestValid(scopeManifest = {}) {
  return scopeManifest.schemaVersion === 1
    && scopeManifest.manifestType === 'p9_protected_scope'
    && REQUIRED_SCOPE_CATEGORIES.every((category) => Array.isArray(scopeManifest.blockingCategories?.[category]?.paths))
    && Array.isArray(scopeManifest.semanticRules);
}

function productionBoundaryPreserved(productionBoundary = {}) {
  return productionBoundary.currentAllowedLevel === 'L0'
    && productionBoundary.productionReady === false
    && productionBoundary.productionAcceptancePassed === false
    && productionBoundary.realPlatformNetworkCalls === 0
    && productionBoundary.realSecretCount === 0
    && productionBoundary.realInventoryWriteApproved === false
    && DISABLED_CAPABILITIES.every((field) => productionBoundary.capabilities?.[field] === false);
}

export function validateP9ProtectedFreezePreparation({
  closure = {},
  closureHead = '',
  currentHead = '',
  currentBranch = '',
  headDetached = false,
  stagedFileCount = 0,
  closureHistoryAvailable = false,
  closureHeadIsAncestor = false,
  validScopeManifest = false,
  protectedChangedFiles = [],
  semanticChanges = [],
  dirtyProtectedChangedFiles = [],
  productionBoundary = {},
} = {}) {
  const closureEvidenceValid = closure.status === 'passed'
    && closure.developmentClosureStatus === 'passed'
    && closure.p9Complete === true
    && closure.developmentClosurePassed === true
    && closure.currentHeadClosureVerified === true
    && closure.currentBranch === 'dev'
    && closure.currentHead === closureHead
    && closure.formalTaskTotal === 38
    && closure.formalTaskCompletedCount === 38
    && closure.formalTaskFailedCount === 0
    && closure.acceptanceCriteriaTotal === 15
    && closure.acceptanceCriteriaPassedCount === 15
    && closure.acceptanceCriteriaFailedCount === 0
    && closure.productionReady === false
    && closure.productionAcceptancePassed === false
    && closure.p10BoundaryPreserved === true;
  const checks = [
    ['closureEvidence', closureEvidenceValid],
    ['currentHead', /^[a-f0-9]{40}$/.test(currentHead)],
    ['currentBranch', currentBranch === 'dev'],
    ['headDetached', headDetached === false],
    ['stagedFileCount', stagedFileCount === 0],
    ['closureHistoryAvailable', closureHistoryAvailable === true],
    ['closureHeadIsAncestor', closureHeadIsAncestor === true],
    ['scopeManifest', validScopeManifest === true],
    ['protectedSourceChanges', protectedChangedFiles.length === 0],
    ['semanticRuleChanges', semanticChanges.length === 0],
    ['dirtyProtectedSource', dirtyProtectedChangedFiles.length === 0],
    ['productionBoundary', productionBoundaryPreserved(productionBoundary)],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    status: failed.length === 0 ? 'passed' : 'blocked',
    failed,
    failedCount: failed.length,
    closureHead,
    currentHead,
    currentBranch,
    protectedChangedFiles,
    semanticChanges,
    dirtyProtectedChangedFiles,
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
  };
}

function collectPreparationInputs() {
  const closure = readJSON(CLOSURE_JSON) || {};
  const scopeManifest = readJSON(P9_PROTECTED_SCOPE_MANIFEST_JSON) || {};
  const productionBoundary = readJSON(P10_DEVELOPMENT_COMPLETION_JSON) || {};
  const closureHead = String(closure.currentClosureHead || closure.currentHead || '');
  const currentHead = git(['rev-parse', 'HEAD']);
  const currentBranch = git(['branch', '--show-current']);
  const closureHistoryAvailable = ensureClosureHistory(closureHead, currentHead);
  const staged = git(['diff', '--cached', '--name-only']);
  const committedChanges = closureHistoryAvailable
    ? parseNameStatus(git(['diff', '--name-status', closureHead, currentHead, '--']))
    : [];
  const workingChanges = parseNameStatus(git(['diff', '--name-status', 'HEAD', '--']));
  const untracked = git(['ls-files', '--others', '--exclude-standard'])
    .split(/\r?\n/).filter(Boolean).map((relativePath) => ({ status: '??', path: normalizePath(relativePath) }));
  const semanticChanges = closureHistoryAvailable ? detectSemanticChanges(closureHead, scopeManifest) : [];
  const dirtySemanticChanges = detectSemanticChanges(currentHead, scopeManifest);
  const protectedScope = classifyP9ProtectedChanges(
    [...committedChanges, ...workingChanges, ...untracked],
    scopeManifest,
    semanticChanges,
  );
  const dirtyScope = classifyP9ProtectedChanges(
    [...workingChanges, ...untracked],
    scopeManifest,
    dirtySemanticChanges,
  );

  return {
    closure,
    closureHead,
    currentHead,
    currentBranch,
    headDetached: git(['rev-parse', '--abbrev-ref', 'HEAD']) === 'HEAD',
    stagedFileCount: staged ? staged.split(/\r?\n/).filter(Boolean).length : 0,
    closureHistoryAvailable,
    closureHeadIsAncestor: closureHistoryAvailable,
    validScopeManifest: scopeManifestValid(scopeManifest),
    protectedChangedFiles: protectedScope.p9ProtectedChangedFiles,
    semanticChanges,
    dirtyProtectedChangedFiles: dirtyScope.p9ProtectedChangedFiles,
    productionBoundary,
  };
}

function run() {
  const preparation = validateP9ProtectedFreezePreparation(collectPreparationInputs());
  if (preparation.status !== 'passed') {
    console.error(JSON.stringify({
      ...preparation,
      freezePath: P9_PROTECTED_SOURCE_FREEZE_JSON,
      freezeWritten: false,
    }, null, 2));
    process.exitCode = 1;
    return;
  }

  const freeze = writeProtectedSourceFreeze();
  const live = computeLiveProtectedSourceManifest();
  const freezeValidation = validateProtectedSourceFreezeBundle({
    freeze,
    live,
    gitState: { currentBranch: preparation.currentBranch, currentHead: preparation.currentHead },
  });
  const status = freezeValidation.status === 'passed' ? 'passed' : 'failed';
  const report = {
    ...preparation,
    status,
    freezePath: P9_PROTECTED_SOURCE_FREEZE_JSON,
    freezeWritten: true,
    protectedSourceManifestSha256: freeze.sha256,
    protectedSourceDriftDetected: freezeValidation.protectedSourceDriftDetected,
    freezeValidationFailed: freezeValidation.failed,
  };
  console.log(JSON.stringify(report, null, 2));
  if (status !== 'passed') process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  run();
}
