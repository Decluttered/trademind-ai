#!/usr/bin/env node
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const backendRoot = path.join(root, 'backend');
const tableNamePattern = /^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$/;
const databaseObjectNamePattern = tableNamePattern;
const phaseNumberPattern = /(?:^|_)p\d+(?:_|$)/u;
const legacyTableSource = new Set([
  path.normalize('backend/internal/database/legacy_schema_names.go'),
  path.normalize('backend/internal/database/legacy_schema_names_test.go'),
  path.normalize('backend/internal/testing/integration/database_migration_test.go'),
]);

function goFiles(directory) {
  const entries = readdirSync(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const fullPath = path.join(directory, entry.name);
    if (entry.isDirectory()) files.push(...goFiles(fullPath));
    else if (entry.isFile() && entry.name.endsWith('.go')) files.push(fullPath);
  }
  return files;
}

function relative(file) {
  return path.normalize(path.relative(root, file));
}

function tableReferences(file, source) {
  const references = [];
  const add = (match, value) => {
    const table = value.split(/\s+/u, 1)[0].replaceAll('"', '').trim();
    if (table) references.push({ table, index: match.index ?? 0 });
  };

  for (const match of source.matchAll(/TableName\(\)\s+string\s*\{\s*return\s+"([^"]+)"/gu)) add(match, match[1]);
  for (const match of source.matchAll(/\bTable\(\s*"([^"]+)"/gu)) add(match, match[1]);
  for (const match of source.matchAll(/\b(?:CREATE\s+TABLE|ALTER\s+TABLE)\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([a-z][a-z0-9_]*)"?/giu)) add(match, match[1]);
  return references;
}

function databaseObjectReferences(source) {
  const references = [];
  const add = (match, value) => {
    if (value) references.push({ name: value, index: match.index ?? 0 });
  };

  for (const match of source.matchAll(/\b(?:index|uniqueIndex|check):([A-Za-z][A-Za-z0-9_]*)/gu)) add(match, match[1]);
  for (const match of source.matchAll(/\b(?:CREATE\s+(?:UNIQUE\s+)?INDEX|ALTER\s+INDEX)\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([A-Za-z][A-Za-z0-9_]*)"?/giu)) add(match, match[1]);
  for (const match of source.matchAll(/\bADD\s+CONSTRAINT\s+"?([A-Za-z][A-Za-z0-9_]*)"?/giu)) add(match, match[1]);
  for (const match of source.matchAll(/\bCREATE\s+TRIGGER\s+(?:IF\s+NOT\s+EXISTS\s+)?"?([A-Za-z][A-Za-z0-9_]*)"?/giu)) add(match, match[1]);
  for (const match of source.matchAll(/\bCREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+"?([A-Za-z][A-Za-z0-9_]*)"?/giu)) add(match, match[1]);
  return references;
}

const violations = [];
const tables = new Map();
const databaseObjects = new Map();
for (const file of goFiles(backendRoot)) {
  const source = readFileSync(file, 'utf8');
  const fileName = relative(file);
  const isTest = fileName.endsWith('_test.go');
  for (const reference of tableReferences(file, source)) {
    const key = reference.table.toLowerCase();
    if (!tables.has(key)) tables.set(key, []);
    tables.get(key).push(fileName);

    if (!tableNamePattern.test(reference.table)) {
      violations.push(`${fileName}: table \"${reference.table}\" must use lowercase snake_case`);
    }
    if (!isTest && !legacyTableSource.has(path.normalize(fileName)) && /^p\d+_/u.test(reference.table)) {
      violations.push(`${fileName}: phase-numbered table \"${reference.table}\" is not allowed in runtime SQL`);
    }
  }
  for (const reference of databaseObjectReferences(source)) {
    const key = reference.name.toLowerCase();
    if (!databaseObjects.has(key)) databaseObjects.set(key, []);
    databaseObjects.get(key).push(fileName);

    if (!databaseObjectNamePattern.test(reference.name)) {
      violations.push(`${fileName}: database object \"${reference.name}\" must use lowercase snake_case`);
    }
    if (!isTest && !legacyTableSource.has(path.normalize(fileName)) && phaseNumberPattern.test(reference.name)) {
      violations.push(`${fileName}: phase-numbered database object \"${reference.name}\" is not allowed in runtime schema`);
    }
  }
}

const legacyImageReferences = [...tables.keys()].filter((table) => table === 'ai_image_task_items');
if (legacyImageReferences.length) {
  const unexpected = [...tables.get('ai_image_task_items')].filter((file) => !legacyTableSource.has(path.normalize(file)));
  if (unexpected.length) violations.push(`legacy image task table is referenced outside migration compatibility code: ${unexpected.join(', ')}`);
}
if (!tables.has('image_task_items')) violations.push('current image task table image_task_items is not referenced by runtime models or SQL');

if (violations.length) {
  console.error('Database naming violations:');
  for (const violation of violations) console.error(`- ${violation}`);
  process.exit(1);
}

console.log(`Database naming check passed (${tables.size} table names and ${databaseObjects.size} database object names scanned).`);
