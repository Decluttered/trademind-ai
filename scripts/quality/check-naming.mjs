#!/usr/bin/env node
import { existsSync, mkdirSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const backend = path.join(root, 'backend');
const goCache = process.env.GOCACHE || path.join(backend, '.gocache');
const goFileNamePattern = /^[a-z][a-z0-9]*(?:_[a-z0-9]+)*\.go$/u;
const goPackageDirectoryPattern = /^[a-z][a-z0-9]*$/u;
const numberedStageTokenPattern = /(?:^|[-_])(?:p\d+|phase\d+|stage\d+|batch\d+)(?=$|[-_.])/iu;
const embeddedPhaseSuffixPattern = /^(?!http2$)[a-z][a-z0-9]*p\d+$/iu;
const temporaryFilePattern = /(?:\.(?:bak|old|orig|tmp)|~)$/iu;

mkdirSync(goCache, { recursive: true });

function run(command, args, env = process.env) {
  const result = spawnSync(command, args, {
    cwd: root,
    env,
    encoding: 'utf8',
    stdio: 'inherit',
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}

function repositoryFiles() {
  const result = spawnSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z'], {
    cwd: root,
    encoding: 'utf8',
  });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(result.stderr || 'git ls-files failed');
  return result.stdout
    .split('\0')
    .filter(Boolean)
    .map((file) => path.join(root, file))
    .filter((file) => existsSync(file));
}

function checkFileNames() {
  const violations = [];
  const files = repositoryFiles();
  for (const file of files) {
    const relative = path.relative(root, file);
    const segments = relative.split(path.sep);
    const fileName = segments.at(-1) || '';

    if (/\s/u.test(relative)) violations.push(`${relative}: whitespace is not allowed in paths`);
    if (temporaryFilePattern.test(fileName)) violations.push(`${relative}: temporary or backup suffix is not allowed`);
    if (fileName.endsWith('.go') && !goFileNamePattern.test(fileName)) {
      violations.push(`${relative}: Go files must use lowercase snake_case`);
    }
    if (fileName.endsWith('.go')) {
      for (const directory of segments.slice(0, -1)) {
        if (!goPackageDirectoryPattern.test(directory)) {
          violations.push(`${relative}: Go package directories must use lowercase names without separators`);
          break;
        }
      }
    }
    for (const segment of segments) {
      const stem = segment.replace(/\.[^.]+$/u, '');
      if (numberedStageTokenPattern.test(segment) || embeddedPhaseSuffixPattern.test(stem)) {
        violations.push(`${relative}: phase or batch numbers are not allowed in file or directory names`);
        break;
      }
    }
  }
  if (violations.length) {
    console.error('File naming violations:');
    for (const violation of violations) console.error(`- ${violation}`);
    process.exit(1);
  }
  console.log(`File naming check passed (${files.length} repository files scanned).`);
}

checkFileNames();
run('go', ['run', './scripts/quality/check_go_naming.go'], { ...process.env, GOCACHE: goCache });
run(process.execPath, ['scripts/quality/check-database-naming.mjs']);
