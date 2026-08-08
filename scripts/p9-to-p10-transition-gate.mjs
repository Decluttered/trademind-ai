import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import {
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  computeLiveProtectedSourceManifest,
  hashProtectedSourceEntries,
} from './p9-protected-source-freeze.mjs';

export const P9_PROTECTED_SCOPE_MANIFEST_JSON = 'docs/p9-protected-scope-manifest.json';
export const P9_TO_P10_TRANSITION_JSON = 'docs/p9-to-p10-transition.json';
export const P9_TO_P10_TRANSITION_MD = 'docs/P9_TO_P10_TRANSITION.md';
export const P9_TO_P10_TRANSITION_GATE_JSON = 'docs/p9-to-p10-transition-gate.json';
export const P9_TO_P10_TRANSITION_GATE_MD = 'docs/P9_TO_P10_TRANSITION_GATE.md';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const CLOSURE_JSON = 'docs/p9-final-development-closure.json';
const PLAN_JSON = 'docs/p9-execution-plan.json';
const POSTGRES_RUNTIME_JSON = 'artifacts/p9-postgres-runtime.json';
const BATCH7_RUNTIME_JSON = 'artifacts/p9-batch7-runtime.json';
const BATCH7_EVIDENCE_JSON = 'docs/p9-task-batch-7-integration-development-closure.json';
const REQUIRED_CATEGORIES = [
  'P9_PRODUCT_SOURCE',
  'P9_SECURITY_AND_PERSISTENCE_CONTRACT',
  'P9_RUNTIME_CONTRACT',
  'P9_GATE_SEMANTICS',
  'P9_ADMIN_PRODUCT_SOURCE',
  'P9_API_CONTRACT',
  'P9_SCOPE_AND_ACCEPTANCE',
];
const REQUIRED_FILES = [
  CLOSURE_JSON,
  PLAN_JSON,
  POSTGRES_RUNTIME_JSON,
  BATCH7_RUNTIME_JSON,
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  BATCH7_EVIDENCE_JSON,
  P9_PROTECTED_SCOPE_MANIFEST_JSON,
];

function rootPath(relativePath) { return path.join(REPO_ROOT, relativePath); }
function read(relativePath) { try { return fs.readFileSync(rootPath(relativePath), 'utf8'); } catch { return ''; } }
function readJSON(relativePath) { try { return JSON.parse(read(relativePath)); } catch { return null; } }
function write(relativePath, value) {
  fs.mkdirSync(path.dirname(rootPath(relativePath)), { recursive: true });
  const content = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  fs.writeFileSync(rootPath(relativePath), content.endsWith('\n') ? content : `${content}\n`, 'utf8');
}
function git(args) {
  try { return execFileSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).trim(); } catch { return ''; }
}
function gitExitZero(args) {
  return spawnSync('git', args, { cwd: REPO_ROOT, encoding: 'utf8' }).status === 0;
}
function sha256(value) { return crypto.createHash('sha256').update(value).digest('hex'); }
function sha256File(relativePath) {
  return fs.existsSync(rootPath(relativePath)) ? sha256(fs.readFileSync(rootPath(relativePath))) : '';
}
function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
}
function sameValue(left, right) { return JSON.stringify(stable(left)) === JSON.stringify(stable(right)); }
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
function changedPath(row) { return normalizePath(typeof row === 'string' ? row : row.path); }

export function hashSourceManifestEntries(entries = []) {
  return hashProtectedSourceEntries(entries);
}

export function classifyP9ProtectedChanges(changedFiles = [], scopeManifest = {}, semanticChanges = []) {
  const categories = Object.fromEntries(REQUIRED_CATEGORIES.map((category) => [category, []]));
  const generatedPatterns = scopeManifest.nonBlockingGeneratedEvidence?.paths || [];
  const blocking = scopeManifest.blockingCategories || {};
  const generatedEvidenceChangedFiles = [];
  const unrelatedChangedFiles = [];

  for (const row of changedFiles) {
    const relativePath = changedPath(row);
    if (!relativePath) continue;
    const matched = [];
    for (const category of REQUIRED_CATEGORIES) {
      if (matchesAny(relativePath, blocking[category]?.paths || [])) matched.push(category);
    }
    if (matched.length > 0) {
      for (const category of matched) categories[category].push(relativePath);
    } else if (matchesAny(relativePath, generatedPatterns)) {
      generatedEvidenceChangedFiles.push(relativePath);
    } else {
      unrelatedChangedFiles.push(relativePath);
    }
  }

  for (const change of semanticChanges) {
    if (categories[change.category]) categories[change.category].push(normalizePath(change.path));
  }
  const semanticPaths = new Set(semanticChanges.map((change) => normalizePath(change.path)));
  for (const category of REQUIRED_CATEGORIES) categories[category] = [...new Set(categories[category])].sort();
  const p9ProtectedChangedFiles = [...new Set(Object.values(categories).flat())].sort();
  return {
    categories,
    p9ProtectedChangedFiles,
    p9ProtectedChangedFileCount: p9ProtectedChangedFiles.length,
    generatedEvidenceChangedFiles: [...new Set(generatedEvidenceChangedFiles)].sort(),
    unrelatedChangedFiles: [...new Set(unrelatedChangedFiles)].filter((file) => !semanticPaths.has(file)).sort(),
  };
}

function scopeManifestValid(manifest = {}) {
  return manifest.schemaVersion === 1
    && manifest.manifestType === 'p9_protected_scope'
    && REQUIRED_CATEGORIES.every((category) => Array.isArray(manifest.blockingCategories?.[category]?.paths));
}

function productPlanValid(plan = {}) {
  const tasks = (plan.workstreams || []).flatMap((workstream) => workstream.tasks || []);
  const productTasks = tasks.filter((task) => task.taskCategory === 'product_implementation' || task.planningFoundation === false);
  const completed = productTasks.filter((task) => task.status === 'completed');
  const acceptanceIds = new Set(productTasks.flatMap((task) => task.acceptanceCriteriaIds || []));
  return plan.phaseStatus === 'Development Complete'
    && plan.executionStatus === 'development_complete'
    && plan.p9Complete === true
    && plan.p9DevelopmentClosurePassed === true
    && productTasks.length === 38
    && completed.length === 38
    && acceptanceIds.size === 15;
}

export function validateP9ToP10TransitionBundle({
  closure = {},
  plan = {},
  postgresRuntime = {},
  batch7Runtime = {},
  batch7Evidence = {},
  sourceManifest = {},
  liveProtectedSourceManifest = {},
  scopeManifest = {},
  gitState = {},
  changedFiles = [],
  semanticChanges = [],
  artifactHashes = {},
  requiredFilesPresent = true,
  commitsSinceP9Closure = [],
} = {}) {
  const p9ClosureHead = String(closure.currentClosureHead || closure.currentHead || '');
  const currentHead = String(gitState.currentHead || '');
  const scope = classifyP9ProtectedChanges(changedFiles, scopeManifest, semanticChanges);
  const manifestPresent = scopeManifestValid(scopeManifest);
  const sourceManifestHead = sourceManifest.gitHead || sourceManifest.currentHead || '';
  const sourceManifestHashValid = sourceManifest.sha256
    && sourceManifest.sha256 === hashSourceManifestEntries(sourceManifest.entries || [])
    && sourceManifestHead === p9ClosureHead
    && sourceManifest.sha256 === closure.protectedSourceManifestSha256
    && sourceManifest.sha256 === closure.postgresRuntimeProtectedSourceManifestSha256
    && sourceManifest.sha256 === closure.batch7RuntimeProtectedSourceManifestSha256
    && sourceManifest.sha256 === postgresRuntime.protectedSourceFreeze?.sha256
    && sourceManifest.sha256 === batch7Runtime.protectedSourceManifestSha256
    && sourceManifest.sha256 === batch7Evidence.runtimeEvidence?.protectedSourceManifestSha256;
  const dirtyProtectedSourceDriftDetected = !sourceManifestHashValid
    || !liveProtectedSourceManifest.sha256
    || liveProtectedSourceManifest.sha256 !== sourceManifest.sha256
    || (liveProtectedSourceManifest.gitHead || liveProtectedSourceManifest.currentHead) !== currentHead;
  const postgresHashValid = Boolean(closure.postgresRuntimeSummarySha256)
    && closure.postgresRuntimeSummarySha256 === artifactHashes.postgresRuntimeSha256;
  const batch7HashValid = Boolean(closure.batch7RuntimeSummarySha256)
    && closure.batch7RuntimeSummarySha256 === artifactHashes.batch7RuntimeSha256;
  const postgresHeadMatches = Boolean(p9ClosureHead)
    && closure.postgresRuntimeHead === p9ClosureHead
    && postgresRuntime.git?.endHead === p9ClosureHead
    && closure.postgresRuntimeRunId === postgresRuntime.runId;
  const batch7HeadMatches = Boolean(p9ClosureHead)
    && closure.batch7RuntimeHead === p9ClosureHead
    && batch7Runtime.currentHead === p9ClosureHead
    && batch7Evidence.currentHead === p9ClosureHead
    && closure.batch7RuntimeRunId === batch7Runtime.runId;
  const closureStatusValid = closure.status === 'passed'
    && closure.developmentClosureStatus === 'passed'
    && closure.p9Complete === true
    && closure.developmentClosurePassed === true
    && closure.formalTaskTotal === 38
    && closure.formalTaskCompletedCount === 38
    && closure.formalTaskFailedCount === 0
    && closure.formalTaskDeferredCount === 0
    && closure.acceptanceCriteriaTotal === 15
    && closure.acceptanceCriteriaPassedCount === 15
    && closure.acceptanceCriteriaFailedCount === 0
    && closure.currentBranch === 'dev'
    && closure.currentHead === p9ClosureHead
    && closure.currentHeadClosureVerified === true
    && productPlanValid(plan);
  const runtimeStatusValid = postgresRuntime.status === 'passed'
    && postgresRuntime.git?.stable === true
    && batch7Runtime.status === 'passed'
    && batch7Runtime.completed === true
    && batch7Evidence.status === 'completed'
    && batch7Evidence.integrationStatus === 'passed';
  const p9FinalClosureArtifactValid = requiredFilesPresent
    && closureStatusValid
    && runtimeStatusValid
    && postgresHeadMatches
    && batch7HeadMatches
    && postgresHashValid
    && batch7HashValid
    && sourceManifestHashValid;
  const productionReadyStillFalse = closure.productionReady === false
    && closure.productionAcceptancePassed === false
    && plan.productionReady === false
    && plan.productionAcceptancePassed === false
    && batch7Runtime.productionReady === false
    && batch7Runtime.productionAcceptancePassed === false;

  const flags = {
    p9ProductSourceChanged: scope.categories.P9_PRODUCT_SOURCE.length > 0,
    p9SecurityContractChanged: scope.categories.P9_SECURITY_AND_PERSISTENCE_CONTRACT.length > 0,
    p9RuntimeContractChanged: scope.categories.P9_RUNTIME_CONTRACT.length > 0,
    p9GateSemanticsChanged: scope.categories.P9_GATE_SEMANTICS.length > 0,
    p9AdminProductSourceChanged: scope.categories.P9_ADMIN_PRODUCT_SOURCE.length > 0,
    p9ApiContractChanged: scope.categories.P9_API_CONTRACT.length > 0,
    p9ScopeChanged: scope.categories.P9_SCOPE_AND_ACCEPTANCE.length > 0,
  };
  const p9ClosureHeadPresent = /^[a-f0-9]{40}$/i.test(p9ClosureHead);
  const p9ClosureReuseEligible = closureStatusValid
    && p9FinalClosureArtifactValid
    && gitState.p9ClosureHeadIsAncestor === true
    && manifestPresent
    && scope.p9ProtectedChangedFileCount === 0
    && dirtyProtectedSourceDriftDetected === false
    && Object.values(flags).every((value) => value === false)
    && productionReadyStillFalse;
  const p9ReclosureRequired = gitState.p9ClosureHeadIsAncestor === true
    && (scope.p9ProtectedChangedFileCount > 0 || dirtyProtectedSourceDriftDetected);
  const minimumRepairAction = p9ReclosureRequired
    ? 'Re-run P9 Current-HEAD Reclosure because protected P9 scope or live source identity changed'
    : gitState.p9ClosureHeadIsAncestor !== true
      ? 'Audit branch divergence or history rewrite before reusing the P9 closure'
      : !p9FinalClosureArtifactValid
        ? 'Repair or regenerate invalid P9 closure evidence before transition'
        : 'none';

  const checks = [
    ['requiredFilesPresent', requiredFilesPresent],
    ['currentBranch', gitState.currentBranch === 'dev'],
    ['headDetached', gitState.headDetached === false],
    ['stagedFileCount', gitState.stagedFileCount === 0],
    ['p9Complete', closure.p9Complete === true],
    ['p9DevelopmentClosurePassed', closure.developmentClosurePassed === true],
    ['p9FinalClosureArtifactPresent', requiredFilesPresent],
    ['p9FinalClosureArtifactValid', p9FinalClosureArtifactValid],
    ['p9ClosureHeadPresent', p9ClosureHeadPresent],
    ['p9ClosureHeadIsAncestor', gitState.p9ClosureHeadIsAncestor === true],
    ['postgresRuntimeHeadMatchesClosureHead', postgresHeadMatches],
    ['batch7RuntimeHeadMatchesClosureHead', batch7HeadMatches],
    ['postgresArtifactSha256Valid', postgresHashValid],
    ['batch7ArtifactSha256Valid', batch7HashValid],
    ['sourceManifestSha256Valid', Boolean(sourceManifestHashValid)],
    ['dirtyProtectedSourceDriftDetected', dirtyProtectedSourceDriftDetected === false],
    ['p9ProtectedScopeManifestPresent', manifestPresent],
    ['p9ProductSourceChanged', flags.p9ProductSourceChanged === false],
    ['p9SecurityContractChanged', flags.p9SecurityContractChanged === false],
    ['p9RuntimeContractChanged', flags.p9RuntimeContractChanged === false],
    ['p9GateSemanticsChanged', flags.p9GateSemanticsChanged === false],
    ['p9AdminProductSourceChanged', flags.p9AdminProductSourceChanged === false],
    ['p9ApiContractChanged', flags.p9ApiContractChanged === false],
    ['p9ScopeChanged', flags.p9ScopeChanged === false],
    ['p9ProtectedChangedFileCount', scope.p9ProtectedChangedFileCount === 0],
    ['p9ClosureReuseEligible', p9ClosureReuseEligible],
    ['productionReady', closure.productionReady === false],
    ['productionAcceptancePassed', closure.productionAcceptancePassed === false],
  ];
  const failed = checks.filter(([, passed]) => !passed).map(([id]) => id);
  return {
    transition: 'P9_TO_P10',
    gate: 'P9-TO-P10-TRANSITION',
    status: failed.length === 0 ? 'passed' : 'blocked',
    checkedAt: new Date().toISOString(),
    failed,
    failedCount: failed.length,
    currentBranch: gitState.currentBranch,
    currentHead,
    headDetached: gitState.headDetached,
    stagedFileCount: gitState.stagedFileCount,
    p9ClosureHead,
    p9ClosureHeadIsAncestor: gitState.p9ClosureHeadIsAncestor === true,
    historicalClosurePassed: closureStatusValid,
    historicalClosureVerified: p9FinalClosureArtifactValid,
    closureIntegrityValid: p9FinalClosureArtifactValid,
    p9FinalClosureArtifactValid,
    postgresRuntimeHead: postgresRuntime.git?.endHead || '',
    batch7RuntimeHead: batch7Runtime.currentHead || '',
    postgresRuntimeMatchesClosureHead: postgresHeadMatches,
    batch7RuntimeMatchesClosureHead: batch7HeadMatches,
    postgresArtifactSha256Valid: postgresHashValid,
    batch7ArtifactSha256Valid: batch7HashValid,
    sourceManifestSha256Valid: Boolean(sourceManifestHashValid),
    closureProtectedSourceManifestSha256: sourceManifest.sha256 || '',
    currentProtectedSourceManifestSha256: liveProtectedSourceManifest.sha256 || '',
    dirtyProtectedSourceDriftDetected,
    protectedScopeManifestSha256: artifactHashes.protectedScopeManifestSha256 || '',
    ...scope,
    ...flags,
    p9ReclosureRequired,
    minimumRepairAction,
    p9ClosureReuseEligible,
    closureReusableForCurrentDescendant: p9ClosureReuseEligible,
    p10PlanningEntryAllowed: p9ClosureReuseEligible,
    p10PlanningPackPrepared: false,
    p10ImplementationStarted: false,
    productionReady: closure.productionReady === true,
    productionAcceptancePassed: closure.productionAcceptancePassed === true,
    commitsSinceP9Closure,
    semanticChanges,
    filesChangedSinceP9Closure: changedFiles.map(changedPath).filter(Boolean),
    checks: checks.map(([id, passed]) => ({ id, status: passed ? 'passed' : 'failed' })),
  };
}

function parseNameStatus(text) {
  return String(text || '').split(/\r?\n/).filter(Boolean).map((line) => {
    const [status, ...paths] = line.split('\t');
    return { status, path: normalizePath(paths.at(-1) || '') };
  }).filter((row) => row.path);
}
function readJSONAt(ref, relativePath) {
  const content = git(['show', `${ref}:${relativePath}`]);
  try { return JSON.parse(content); } catch { return null; }
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
function detectSemanticChanges(closureHead, currentHead, scopeManifest) {
  const changes = [];
  for (const rule of scopeManifest.semanticRules || []) {
    const before = semanticValue(readJSONAt(closureHead, rule.path), rule);
    const after = semanticValue(readJSONAt(currentHead, rule.path), rule);
    if (!sameValue(before, after)) changes.push({ id: rule.id, path: rule.path, category: rule.category });
  }
  return changes;
}
function renderTransitionMarkdown(report) {
  const outcome = report.status === 'passed' ? 'Passed' : 'Blocked';
  return `# P9 to P10 Transition\n\nStatus: **${outcome}**\n\nThis evidence validates whether the P9 development closure can be reused by the current HEAD and live protected-source identity. It does not start P10 planning.\n\n\`\`\`text\ntransition=P9_TO_P10\np9ClosureHead=${report.p9ClosureHead}\ncurrentHead=${report.currentHead}\np9ClosureHeadIsAncestor=${report.p9ClosureHeadIsAncestor}\nhistoricalClosureVerified=${report.historicalClosureVerified}\nclosureIntegrityValid=${report.closureIntegrityValid}\np9ProtectedChangedFileCount=${report.p9ProtectedChangedFileCount}\ndirtyProtectedSourceDriftDetected=${report.dirtyProtectedSourceDriftDetected}\np9ReclosureRequired=${report.p9ReclosureRequired}\np9ClosureReuseEligible=${report.p9ClosureReuseEligible}\np10PlanningEntryAllowed=${report.p10PlanningEntryAllowed}\nproductionReady=${report.productionReady}\nproductionAcceptancePassed=${report.productionAcceptancePassed}\n\`\`\`\n\n## Commits Since Closure\n\n${report.commitsSinceP9Closure.length ? report.commitsSinceP9Closure.map((item) => `- \`${item.commit}\` ${item.subject}`).join('\n') : '- None'}\n\n## Protected Changes\n\n${report.p9ProtectedChangedFiles.length ? report.p9ProtectedChangedFiles.map((file) => `- \`${file}\``).join('\n') : '- None'}\n\n## Decision\n\nMinimum repair action: ${report.minimumRepairAction}.\n\n## Boundary\n\nP10 planning is not started by this gate. Real provider, OAuth, platform network access, inventory mutation, worker, retry, gray release, tag, and release remain disabled or deferred.\n`;
}
function renderGateMarkdown(report) {
  return `# P9 to P10 Transition Gate\n\nStatus: **${report.status}**\n\n- P9 closure HEAD: ${report.p9ClosureHead}\n- Current HEAD: ${report.currentHead}\n- Closure is ancestor: ${report.p9ClosureHeadIsAncestor}\n- Protected files changed: ${report.p9ProtectedChangedFileCount}\n- Dirty protected source drift detected: ${report.dirtyProtectedSourceDriftDetected}\n- Reclosure required: ${report.p9ReclosureRequired}\n- P10 planning entry allowed: ${report.p10PlanningEntryAllowed}\n- Failed checks: ${report.failedCount ? report.failed.join(', ') : 'none'}\n\n${report.checks.map((check) => `- ${check.status === 'passed' ? 'PASS' : 'FAIL'} \`${check.id}\``).join('\n')}\n`;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  const closure = readJSON(CLOSURE_JSON) || {};
  const plan = readJSON(PLAN_JSON) || {};
  const postgresRuntime = readJSON(POSTGRES_RUNTIME_JSON) || {};
  const batch7Runtime = readJSON(BATCH7_RUNTIME_JSON) || {};
  const batch7Evidence = readJSON(BATCH7_EVIDENCE_JSON) || {};
  const sourceManifest = readJSON(P9_PROTECTED_SOURCE_FREEZE_JSON) || {};
  const liveProtectedSourceManifest = computeLiveProtectedSourceManifest();
  const scopeManifest = readJSON(P9_PROTECTED_SCOPE_MANIFEST_JSON) || {};
  const currentBranch = git(['branch', '--show-current']);
  const currentHead = git(['rev-parse', 'HEAD']);
  const p9ClosureHead = String(closure.currentClosureHead || closure.currentHead || '');
  const changedFiles = p9ClosureHead ? parseNameStatus(git(['diff', '--name-status', `${p9ClosureHead}..${currentHead}`])) : [];
  const commitsSinceP9Closure = p9ClosureHead
    ? git(['log', '--format=%H%x09%s', `${p9ClosureHead}..${currentHead}`]).split(/\r?\n/).filter(Boolean).map((line) => {
      const [commit, ...subject] = line.split('\t');
      return { commit, subject: subject.join('\t') };
    })
    : [];
  const staged = git(['diff', '--cached', '--name-only']);
  const report = validateP9ToP10TransitionBundle({
    closure,
    plan,
    postgresRuntime,
    batch7Runtime,
    batch7Evidence,
    sourceManifest,
    liveProtectedSourceManifest,
    scopeManifest,
    gitState: {
      currentBranch,
      currentHead,
      headDetached: git(['rev-parse', '--abbrev-ref', 'HEAD']) === 'HEAD',
      stagedFileCount: staged ? staged.split(/\r?\n/).filter(Boolean).length : 0,
      p9ClosureHeadIsAncestor: Boolean(p9ClosureHead) && gitExitZero(['merge-base', '--is-ancestor', p9ClosureHead, currentHead]),
    },
    changedFiles,
    semanticChanges: p9ClosureHead ? detectSemanticChanges(p9ClosureHead, currentHead, scopeManifest) : [],
    artifactHashes: {
      postgresRuntimeSha256: sha256File(POSTGRES_RUNTIME_JSON),
      batch7RuntimeSha256: sha256File(BATCH7_RUNTIME_JSON),
      protectedScopeManifestSha256: sha256File(P9_PROTECTED_SCOPE_MANIFEST_JSON),
    },
    requiredFilesPresent: REQUIRED_FILES.every((relativePath) => fs.existsSync(rootPath(relativePath))),
    commitsSinceP9Closure,
  });
  write(P9_TO_P10_TRANSITION_JSON, report);
  write(P9_TO_P10_TRANSITION_MD, renderTransitionMarkdown(report));
  write(P9_TO_P10_TRANSITION_GATE_JSON, report);
  write(P9_TO_P10_TRANSITION_GATE_MD, renderGateMarkdown(report));
  console.log(JSON.stringify(report, null, 2));
  if (report.status !== 'passed') process.exitCode = 1;
}
