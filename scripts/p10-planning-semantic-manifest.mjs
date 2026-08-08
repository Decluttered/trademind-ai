import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

export const P10_PLANNING_SEMANTIC_FILES = [
  'docs/p10-planning-pack.json',
  'docs/p10-owner-decision-proposal.json',
  'docs/p10-owner-scope-decision.json',
  'docs/p10-production-boundary.json',
  'docs/p10-execution-plan-draft.json',
  'docs/p10-execution-plan.json',
  'docs/p10-acceptance-criteria-draft.json',
  'docs/p10-acceptance-criteria.json',
  'docs/p10-risk-register.json',
  'scripts/p10-planning-semantic-manifest.mjs',
  'scripts/p10-planning-pack-gate.mjs',
  'scripts/p10-owner-decision-gate.mjs',
];

function sha256(value) {
  return crypto.createHash('sha256').update(value).digest('hex');
}

export function computeP10PlanningSemanticManifest({
  repoRoot = REPO_ROOT,
  files = P10_PLANNING_SEMANTIC_FILES,
} = {}) {
  const entries = files.map((relativePath) => {
    const absolutePath = path.join(repoRoot, relativePath);
    const content = fs.readFileSync(absolutePath);
    return { path: relativePath, sha256: sha256(content) };
  });
  const canonical = JSON.stringify({ schemaVersion: 1, entries });
  return {
    schemaVersion: 1,
    sha256: sha256(canonical),
    fileCount: entries.length,
    entries,
  };
}
