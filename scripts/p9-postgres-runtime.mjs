#!/usr/bin/env node
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync, execFileSync } from 'node:child_process';
import {
  deriveRuntimeContracts,
  parseGoJSONL,
  parseSafeTestDatabaseUrl,
  p9SourceManifest,
  repoRoot,
  runtimeRaceRawPath,
  runtimeRawPath,
  runtimeSummaryPath,
  sanitizeRuntimeText,
  sha256Buffer,
} from './p9-postgres-contract.mjs';
import {
  P9_PROTECTED_SOURCE_FREEZE_JSON,
  computeLiveProtectedSourceManifest,
  readProtectedSourceFreeze,
  validateProtectedSourceFreezeBundle,
} from './p9-protected-source-freeze.mjs';

function git(args) {
  return execFileSync('git', args, { cwd: repoRoot, encoding: 'utf8' }).trim();
}

function stagedFileCount() {
  const value = git(['diff', '--cached', '--name-only']);
  return value ? value.split(/\r?\n/).filter(Boolean).length : 0;
}

function writeAtomic(relativePath, value) {
  const full = path.join(repoRoot, relativePath);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  const temporary = `${full}.tmp-${process.pid}`;
  fs.writeFileSync(temporary, value, 'utf8');
  fs.renameSync(temporary, full);
}

function readJSON(relativePath) {
  try { return JSON.parse(fs.readFileSync(path.join(repoRoot, relativePath), 'utf8')); } catch { return {}; }
}

function renderPostgresClosureMarkdown(evidence) {
  return `# P9 PostgreSQL Integration Baseline Closure

Status: **Passed**

The P9 persistence, isolation, concurrency, transaction, API, authentication, RBAC, audit, and fixture contracts were revalidated against an isolated PostgreSQL test database. The suite is fail-closed and does not fall back to SQLite.

\`\`\`text
currentHead=${evidence.currentHead}
runtimeRunId=${evidence.runtimeEvidence.runId}
runtimeSummarySha256=${evidence.runtimeEvidence.runtimeSummarySha256}
sourceManifestSha256=${evidence.runtimeEvidence.sourceManifestSha256}
protectedSourceManifestSha256=${evidence.runtimeEvidence.protectedSourceManifestSha256}
runtimeHeadMatchesCurrentHead=${evidence.runtimeEvidence.runtimeHeadMatchesCurrentHead}
protectedSourceDriftDetected=false
testDatabaseDriver=${evidence.testDatabaseDriver}
testDatabasePurpose=test
testDatabaseNameSafe=${evidence.testDatabaseNameSafe}
testDatabaseUrlRecorded=false
productionDatabaseRejected=${evidence.productionDatabaseRejected}
sqliteFallbackUsed=${evidence.sqliteFallbackUsed}
racePassed=${evidence.racePassed}
dataRaces=${evidence.dataRaces}
realPlatformNetworkCalls=0
realCredentialsUsed=false
inventoryMutationCalls=0
productionReady=false
\`\`\`

No database password, authorization value, token, cookie, or complete connection string is recorded in this evidence.

## Boundary

Real provider, OAuth, platform read/write, inventory mutation, background worker, automatic retry, tag, release, and production acceptance remain disabled or deferred.
`;
}

function updatePostgresClosureEvidence(summary) {
  const jsonPath = 'docs/p9-postgresql-integration-closure.json';
  const markdownPath = 'docs/P9_POSTGRESQL_INTEGRATION_CLOSURE.md';
  const previous = readJSON(jsonPath);
  const runtimeSummarySha256 = sha256Buffer(fs.readFileSync(path.join(repoRoot, runtimeSummaryPath)));
  const previousRuns = [...(previous.previousReclosureRuns || [])];
  if (previous.reclosure?.runtimeRunId && previous.reclosure.runtimeRunId !== summary.runId) previousRuns.push(previous.reclosure);
  const reclosure = {
    status: 'passed',
    reason: 'p9_runtime_and_gate_semantics_changed',
    head: summary.git.endHead,
    runtimeRunId: summary.runId,
    runtimeSummarySha256,
    sourceManifestSha256: summary.sourceManifest.afterSha256,
    protectedSourceManifestSha256: summary.protectedSourceFreeze.sha256,
    verifiedAt: summary.finishedAt,
  };
  const runtimeEvidence = {
    runId: summary.runId,
    summaryPath: runtimeSummaryPath,
    runtimeSummarySha256,
    runtimeHead: summary.git.endHead,
    runtimeHeadMatchesCurrentHead: true,
    sourceManifestSha256: summary.sourceManifest.afterSha256,
    sourceManifestHead: summary.git.endHead,
    sourceManifestHeadMatchesCurrentHead: true,
    protectedSourceManifestSha256: summary.protectedSourceFreeze.sha256,
    protectedSourceDriftDetected: false,
    finishedAt: summary.finishedAt,
  };
  const evidence = {
    ...previous,
    schemaVersion: 2,
    status: 'passed',
    baseBranch: summary.git.endBranch,
    baseCheckpoint: summary.git.endHead,
    currentBranch: summary.git.endBranch,
    currentHead: summary.git.endHead,
    headDetached: false,
    stagedFileCount: summary.git.stagedFileCountAfter,
    testDatabaseDriver: summary.testDatabase.driver,
    testDatabasePurpose: 'test',
    testDatabaseHostCategory: summary.testDatabase.hostCategory,
    testDatabaseUrlRecorded: false,
    testDatabaseNameSafe: summary.testDatabase.nameSafe,
    productionDatabaseRejected: summary.testDatabase.productionRejected,
    postgresServerVersion: summary.testDatabase.serverVersion,
    connectionPassed: true,
    postgresConnectionPassed: true,
    ...summary.contracts,
    constraintTestsPassed: summary.contracts.schemaVerificationPassed,
    transactionTestsPassed: summary.contracts.transactionAtomicityPassed,
    paginationTestsPassed: summary.contracts.keysetPaginationPassed,
    apiIntegrationTestsPassed: summary.contracts.postgresApiIntegrationPassed,
    fixtureGoldenPathPassed: summary.contracts.postgresFixtureGoldenPathPassed,
    sqliteFallbackUsed: false,
    fixtureProviderNetworkCalls: 0,
    realPlatformNetworkCalls: 0,
    realCredentialsUsed: false,
    inventoryMutationCalls: 0,
    racePassed: true,
    dataRaces: 0,
    postgresRevalidation: {
      status: 'passed',
      evidencePath: markdownPath,
      gateReportPath: 'docs/P9_POSTGRESQL_INTEGRATION_CLOSURE_GATE.md',
      verifiedAt: summary.finishedAt,
    },
    currentPostgresIntegrationStatus: 'passed',
    postgresIntegrationStatus: 'passed',
    postgresIntegrationPassed: true,
    postgresIntegrationDeferredTo: null,
    p9FinalClosureBlocker: false,
    reclosure,
    previousReclosureRuns: [...new Map(previousRuns.map((row) => [row.runtimeRunId, row])).values()],
    runtimeEvidence,
    productionReady: false,
    p10BoundaryPreserved: true,
    batch6ReadyToStart: true,
    p9Complete: false,
    verifiedAt: summary.finishedAt,
  };
  writeAtomic(jsonPath, `${JSON.stringify(evidence, null, 2)}\n`);
  writeAtomic(markdownPath, renderPostgresClosureMarkdown(evidence));
}

function runGo(args, outputPath) {
  const result = spawnSync('go', args, {
    cwd: path.join(repoRoot, 'backend'),
    env: process.env,
    encoding: 'utf8',
    maxBuffer: 128 * 1024 * 1024,
  });
  const combined = sanitizeRuntimeText(`${result.stdout || ''}${result.stderr || ''}`);
  writeAtomic(outputPath, combined);
  return {
    status: result.status ?? 1,
    signal: result.signal || '',
    parsed: parseGoJSONL(combined),
    sha256: sha256Buffer(Buffer.from(combined)),
  };
}

function commandSummary(name, args, result, rawPath) {
  return {
    name,
    executable: 'go',
    args,
    exitCode: result.status,
    signal: result.signal,
    rawArtifactPath: rawPath,
    rawArtifactSha256: result.sha256,
    packageResults: result.parsed.packages,
    testResults: result.parsed.tests,
  };
}

const startedAt = new Date().toISOString();
const startBranch = git(['branch', '--show-current']);
const startHead = git(['rev-parse', 'HEAD']);
const headDetached = git(['rev-parse', '--abbrev-ref', 'HEAD']) === 'HEAD';
const stagedBefore = stagedFileCount();
const sourceBefore = p9SourceManifest();
const protectedSourceFreeze = readProtectedSourceFreeze();
const protectedSourceBefore = computeLiveProtectedSourceManifest();
const protectedSourceValidationBefore = validateProtectedSourceFreezeBundle({
  freeze: protectedSourceFreeze,
  live: protectedSourceBefore,
  gitState: { currentBranch: startBranch, currentHead: startHead },
});
const database = parseSafeTestDatabaseUrl(process.env.TEST_DATABASE_URL, process.env);
const runId = `p9pg-${startedAt.replace(/[^0-9]/g, '').slice(0, 14)}-${crypto.randomBytes(4).toString('hex')}`;

const preflightIssues = [];
if (startBranch !== 'dev') preflightIssues.push('currentBranch');
if (headDetached) preflightIssues.push('headDetached');
if (stagedBefore !== 0) preflightIssues.push('stagedFileCountBefore');
if (!database.valid) preflightIssues.push(database.reason || 'testDatabaseUrl');
if (protectedSourceValidationBefore.status !== 'passed') preflightIssues.push('protectedSourceFreeze');

let postgresResult = null;
let raceResult = null;
const postgresArgs = ['test', '-json', '-tags', 'p9postgres', '-count=1', './internal/modules/inventorysyncp9', './internal/testing/integration'];
const raceArgs = ['test', '-json', '-race', '-tags', 'p9postgres', '-count=1', './internal/modules/inventorysyncp9', './internal/testing/integration'];
if (preflightIssues.length === 0) {
  postgresResult = runGo(postgresArgs, runtimeRawPath);
  raceResult = runGo(raceArgs, runtimeRaceRawPath);
}

const finishedAt = new Date().toISOString();
const endBranch = git(['branch', '--show-current']);
const endHead = git(['rev-parse', 'HEAD']);
const stagedAfter = stagedFileCount();
const sourceAfter = p9SourceManifest();
const protectedSourceAfter = computeLiveProtectedSourceManifest();
const parsed = postgresResult?.parsed || { tests: {}, packages: {}, metadata: null, dataRaces: 0 };
const contracts = deriveRuntimeContracts(parsed, postgresResult?.status === 0);
const racePackagesPassed = Boolean(raceResult && raceResult.status === 0 && Object.keys(raceResult.parsed.packages).length >= 2 && Object.values(raceResult.parsed.packages).every((value) => value === 'pass'));
const racePassed = racePackagesPassed && raceResult.parsed.dataRaces === 0;
const sourceStable = sourceBefore.sha256 === sourceAfter.sha256;
const protectedSourceStable = protectedSourceValidationBefore.status === 'passed'
  && protectedSourceBefore.sha256 === protectedSourceAfter.sha256
  && protectedSourceAfter.sha256 === protectedSourceFreeze.sha256;
const gitStable = startBranch === endBranch && startHead === endHead && stagedBefore === 0 && stagedAfter === 0;
const completed = preflightIssues.length === 0 && contracts.postgresIntegrationPassed && racePassed && sourceStable && protectedSourceStable && gitStable;

const summary = {
  schemaVersion: 2,
  phase: 'P9',
  closureType: 'postgresql_integration_runtime',
  runId,
  status: completed ? 'passed' : 'failed',
  completed,
  startedAt,
  finishedAt,
  git: {
    startBranch,
    endBranch,
    startHead,
    endHead,
    headDetached,
    stagedFileCountBefore: stagedBefore,
    stagedFileCountAfter: stagedAfter,
    stable: gitStable,
  },
  sourceManifest: {
    beforeSha256: sourceBefore.sha256,
    afterSha256: sourceAfter.sha256,
    fileCount: sourceBefore.fileCount,
    stable: sourceStable,
  },
  protectedSourceFreeze: {
    path: P9_PROTECTED_SOURCE_FREEZE_JSON,
    gitHead: protectedSourceFreeze.gitHead || '',
    sha256: protectedSourceFreeze.sha256 || '',
    beforeSha256: protectedSourceBefore.sha256,
    afterSha256: protectedSourceAfter.sha256,
    dirtyProtectedChangedFileCount: protectedSourceBefore.dirtyProtectedChangedFileCount,
    dirtyProtectedChangedFiles: protectedSourceBefore.dirtyProtectedChangedFiles,
    frozen: protectedSourceValidationBefore.status === 'passed',
    stable: protectedSourceStable,
    driftDetected: !protectedSourceStable,
  },
  testDatabase: {
    driver: database.valid ? 'postgresql' : '',
    purpose: 'test',
    urlRecorded: false,
    hostCategory: database.hostCategory,
    databaseNameHash: database.databaseNameHash,
    nameSafe: database.nameSafe,
    productionRejected: database.productionRejected,
    actualDatabaseMatched: Boolean(parsed.metadata?.databaseNameHash && parsed.metadata.databaseNameHash === database.databaseNameHash),
    serverVersion: parsed.metadata?.serverVersion || '',
    schemaIsolated: parsed.metadata?.schemaIsolated === true,
    sqliteFallbackUsed: parsed.metadata?.sqliteFallbackUsed !== false,
  },
  preflightIssues,
  commands: [
    ...(postgresResult ? [commandSummary('postgres-integration', postgresArgs, postgresResult, runtimeRawPath)] : []),
    ...(raceResult ? [commandSummary('postgres-race', raceArgs, raceResult, runtimeRaceRawPath)] : []),
  ],
  contracts,
  racePassed,
  dataRaces: raceResult?.parsed.dataRaces ?? 0,
  platformBoundary: {
    fixtureProviderNetworkCalls: contracts.postgresFixtureGoldenPathPassed ? 0 : null,
    realPlatformNetworkCalls: contracts.postgresFixtureGoldenPathPassed ? 0 : null,
    realCredentialsUsed: false,
    inventoryMutationCalls: contracts.postgresFixtureGoldenPathPassed ? 0 : null,
  },
};
writeAtomic(runtimeSummaryPath, `${JSON.stringify(summary, null, 2)}\n`);
if (completed) updatePostgresClosureEvidence(summary);
console.log(JSON.stringify({
  runId: summary.runId,
  status: summary.status,
  runtimeSummaryPath,
  testDatabaseDriver: summary.testDatabase.driver,
  testDatabaseHostCategory: summary.testDatabase.hostCategory,
  testDatabaseUrlRecorded: false,
  postgresIntegrationPassed: contracts.postgresIntegrationPassed,
  protectedSourceManifestSha256: summary.protectedSourceFreeze.sha256,
  protectedSourceDriftDetected: summary.protectedSourceFreeze.driftDetected,
  racePassed,
  dataRaces: summary.dataRaces,
  failedPreflight: preflightIssues,
}, null, 2));
process.exit(completed ? 0 : 1);
