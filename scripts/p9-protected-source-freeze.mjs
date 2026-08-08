#!/usr/bin/env node
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const P9_PROTECTED_SCOPE_MANIFEST_JSON = 'docs/p9-protected-scope-manifest.json';
export const P9_PROTECTED_SOURCE_FREEZE_JSON = 'artifacts/p9-protected-source-freeze.json';

export const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const CATEGORY_MAP = {
  P9_PRODUCT_SOURCE: 'product_source',
  P9_SECURITY_AND_PERSISTENCE_CONTRACT: 'persistence_security',
  P9_RUNTIME_CONTRACT: 'runtime_contract',
  P9_GATE_SEMANTICS: 'gate_semantics',
  P9_ADMIN_PRODUCT_SOURCE: 'admin_product_source',
  P9_API_CONTRACT: 'api_contract',
  P9_SCOPE_AND_ACCEPTANCE: 'gate_semantics',
};

function rootPath(relativePath) { return path.join(repoRoot, relativePath); }
function git(args) { return execFileSync('git', args, { cwd: repoRoot, encoding: 'utf8' }).trim(); }
function normalizePath(value) { return String(value || '').replaceAll('\\', '/').replace(/^\.\//, ''); }
function escapeRegExp(value) { return value.replace(/[|\\{}()[\]^$+?.]/g, '\\$&'); }
function globRegExp(pattern) {
  const token = normalizePath(pattern);
  let source = '';
  for (let index = 0; index < token.length; index += 1) {
    if (token[index] === '*' && token[index + 1] === '*') {
      source += '.*';
      index += 1;
    } else if (token[index] === '*') {
      source += '[^/]*';
    } else {
      source += escapeRegExp(token[index]);
    }
  }
  return new RegExp(`^${source}$`);
}
function matchesAny(relativePath, patterns = []) {
  const normalized = normalizePath(relativePath);
  return patterns.some((pattern) => globRegExp(pattern).test(normalized));
}
function sha256(value) { return crypto.createHash('sha256').update(value).digest('hex'); }
function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
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
function readJSON(relativePath) {
  try { return JSON.parse(fs.readFileSync(rootPath(relativePath), 'utf8')); } catch { return null; }
}
function listRepositoryFiles() {
  const output = git(['ls-files', '--cached', '--others', '--exclude-standard']);
  return output.split(/\r?\n/).filter(Boolean).map(normalizePath).sort();
}
function listDirtyFiles() {
  const changed = git(['diff', '--name-only']).split(/\r?\n/).filter(Boolean);
  const untracked = git(['ls-files', '--others', '--exclude-standard']).split(/\r?\n/).filter(Boolean);
  return [...new Set([...changed, ...untracked].map(normalizePath))].sort();
}

export function hashProtectedSourceEntries(entries = []) {
  const hash = crypto.createHash('sha256');
  for (const entry of [...entries].sort((left, right) => left.path.localeCompare(right.path))) {
    hash.update(normalizePath(entry.path));
    hash.update('\0');
    hash.update(String(entry.hashKind || 'file'));
    hash.update('\0');
    hash.update([...(entry.categories || [entry.category])].filter(Boolean).sort().join(','));
    hash.update('\0');
    hash.update(String(entry.sha256 || ''));
    hash.update('\n');
  }
  return hash.digest('hex');
}

export function computeLiveProtectedSourceManifest({ scopeManifest: injectedScopeManifest } = {}) {
  const scopeManifest = injectedScopeManifest || readJSON(P9_PROTECTED_SCOPE_MANIFEST_JSON) || {};
  const files = listRepositoryFiles();
  const generatedPatterns = scopeManifest.nonBlockingGeneratedEvidence?.paths || [];
  const categoryPatterns = Object.entries(scopeManifest.blockingCategories || {});
  const entries = [];

  for (const relativePath of files) {
    if (matchesAny(relativePath, generatedPatterns)) continue;
    const categories = categoryPatterns
      .filter(([, value]) => matchesAny(relativePath, value.paths || []))
      .map(([category]) => CATEGORY_MAP[category])
      .filter(Boolean);
    if (categories.length === 0) continue;
    const content = fs.readFileSync(rootPath(relativePath));
    const uniqueCategories = [...new Set(categories)].sort();
    entries.push({
      path: relativePath,
      category: uniqueCategories[0],
      categories: uniqueCategories,
      hashKind: 'file',
      sha256: sha256(content),
    });
  }

  for (const rule of scopeManifest.semanticRules || []) {
    if (matchesAny(rule.path, generatedPatterns)) continue;
    const value = semanticValue(readJSON(rule.path), rule);
    if (value === null) continue;
    const category = CATEGORY_MAP[rule.category] || 'gate_semantics';
    entries.push({
      path: normalizePath(rule.path),
      category,
      categories: [category],
      hashKind: `semantic:${rule.kind}`,
      semanticRuleId: rule.id,
      sha256: sha256(JSON.stringify(stable(value))),
    });
  }

  entries.sort((left, right) => left.path.localeCompare(right.path) || left.hashKind.localeCompare(right.hashKind));
  const protectedPaths = new Set(entries.map((entry) => entry.path));
  const dirtyProtectedChangedFiles = listDirtyFiles().filter((relativePath) => protectedPaths.has(relativePath));
  return {
    schemaVersion: 1,
    manifestType: 'p9_protected_source_freeze',
    phase: 'P9',
    gitHead: git(['rev-parse', 'HEAD']),
    currentBranch: git(['branch', '--show-current']),
    scopeManifestPath: P9_PROTECTED_SCOPE_MANIFEST_JSON,
    scopeManifestSha256: fs.existsSync(rootPath(P9_PROTECTED_SCOPE_MANIFEST_JSON))
      ? sha256(fs.readFileSync(rootPath(P9_PROTECTED_SCOPE_MANIFEST_JSON)))
      : '',
    sha256: hashProtectedSourceEntries(entries),
    fileCount: entries.length,
    dirtyProtectedChangedFileCount: dirtyProtectedChangedFiles.length,
    dirtyProtectedChangedFiles,
    entries,
  };
}

export function readProtectedSourceFreeze() {
  return readJSON(P9_PROTECTED_SOURCE_FREEZE_JSON) || {};
}

export function validateProtectedSourceFreezeBundle({ freeze = {}, live = {}, gitState = {} } = {}) {
  const currentHead = gitState.currentHead || live.gitHead || '';
  const currentBranch = gitState.currentBranch || live.currentBranch || '';
  const checks = [
    ['manifestType', freeze.manifestType === 'p9_protected_source_freeze'],
    ['currentBranch', freeze.currentBranch === currentBranch && currentBranch === 'dev'],
    ['gitHead', Boolean(currentHead) && freeze.gitHead === currentHead && live.gitHead === currentHead],
    ['manifestHash', /^[a-f0-9]{64}$/.test(freeze.sha256 || '') && freeze.sha256 === hashProtectedSourceEntries(freeze.entries || [])],
    ['liveManifestHash', freeze.sha256 === live.sha256],
    ['fileCount', freeze.fileCount === (freeze.entries || []).length && freeze.fileCount === live.fileCount],
    ['dirtyProtectedFilesIncluded', (live.dirtyProtectedChangedFiles || []).every((file) => (freeze.entries || []).some((entry) => entry.path === file))],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    status: failed.length === 0 ? 'passed' : 'failed',
    failed,
    failedCount: failed.length,
    protectedSourceFrozen: failed.length === 0,
    protectedSourceManifestSha256: freeze.sha256 || '',
    currentProtectedSourceManifestSha256: live.sha256 || '',
    protectedSourceDriftDetected: freeze.sha256 !== live.sha256,
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
  };
}

export function writeProtectedSourceFreeze() {
  const manifest = { ...computeLiveProtectedSourceManifest(), generatedAt: new Date().toISOString() };
  const target = rootPath(P9_PROTECTED_SOURCE_FREEZE_JSON);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  const temporary = `${target}.tmp-${process.pid}`;
  fs.writeFileSync(temporary, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  fs.renameSync(temporary, target);
  return manifest;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  const write = process.argv.includes('--write');
  const live = computeLiveProtectedSourceManifest();
  const freeze = write ? writeProtectedSourceFreeze() : readProtectedSourceFreeze();
  const validation = validateProtectedSourceFreezeBundle({ freeze, live: write ? computeLiveProtectedSourceManifest() : live });
  console.log(JSON.stringify({
    freezePath: P9_PROTECTED_SOURCE_FREEZE_JSON,
    gitHead: freeze.gitHead || live.gitHead,
    protectedSourceFrozen: validation.protectedSourceFrozen,
    protectedSourceManifestSha256: freeze.sha256 || '',
    currentProtectedSourceManifestSha256: live.sha256,
    protectedSourceDriftDetected: validation.protectedSourceDriftDetected,
    dirtyProtectedChangedFileCount: live.dirtyProtectedChangedFileCount,
    dirtyProtectedChangedFiles: live.dirtyProtectedChangedFiles,
    failed: validation.failed,
  }, null, 2));
  if (validation.status !== 'passed') process.exitCode = 1;
}
