import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { readEnvFile, validatePreproductionContract } from './preproduction-contract.mjs';

function argument(name, fallback = '') {
  const index = process.argv.indexOf(name);
  return index >= 0 && process.argv[index + 1] ? process.argv[index + 1] : fallback;
}

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const envFile = path.resolve(repoRoot, '.env');
const mode = argument('--mode', 'startup');
const allowedModes = ['config', 'startup', 'migration', 'backup', 'restore', 'rollback', 'teardown'];
const report = validatePreproductionContract({ ...readEnvFile(envFile), ...process.env });
if (!allowedModes.includes(mode)) report.failed.push('operationMode');
report.mode = mode;
report.envFile = path.relative(repoRoot, envFile).replaceAll('\\', '/');
report.failedCount = report.failed.length;
report.status = report.failedCount === 0 ? 'passed' : 'failed';
console.log(JSON.stringify(report, null, 2));
if (report.failedCount > 0) process.exitCode = 1;
