import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { sanitizeProfileKey } from './profile-key.js';

/** Collector package root directory (independent of the runtime cwd). */
export const COLLECTOR_PACKAGE_ROOT = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../..',
);

/**
 * Root directory for persistent profiles of providers such as 1688.
 * Prefers COLLECTOR_BROWSER_PROFILE_DIR, then BROWSER_PROFILE_ROOT, defaulting to collector/data/browser-profiles.
 */
export function getBrowserProfileRoot(): string {
  const raw =
    process.env.COLLECTOR_BROWSER_PROFILE_DIR?.trim() ||
    process.env.COLLECTOR_PROFILE_DIR?.trim() ||
    process.env.BROWSER_PROFILE_ROOT?.trim() ||
    '';
  if (raw) {
    return path.isAbsolute(raw) ? raw : path.resolve(COLLECTOR_PACKAGE_ROOT, raw);
  }
  return path.join(COLLECTOR_PACKAGE_ROOT, 'data', 'browser-profiles');
}

/** 1688 userDataDir: BROWSER_PROFILE_ROOT/1688 */
export function get1688UserDataDir(): string {
  return path.join(getBrowserProfileRoot(), '1688');
}

/** Pinduoduo-dedicated profile (isolated from 1688 / custom): BROWSER_PROFILE_ROOT/pinduoduo */
export function getPinduoduoUserDataDir(): string {
  return path.join(getBrowserProfileRoot(), 'pinduoduo');
}

/** Taobao/Tmall-dedicated profile: BROWSER_PROFILE_ROOT/taobao_tmall */
export function getTaobaoTmallUserDataDir(): string {
  return path.join(getBrowserProfileRoot(), 'taobao_tmall');
}

/** Custom collect browser profile: BROWSER_PROFILE_ROOT/custom/{profileKey} */
export function getCustomProfileUserDataDir(profileKey: string): string {
  const safe = sanitizeProfileKey(profileKey);
  return path.join(getBrowserProfileRoot(), 'custom', safe);
}

export function getStorageStateRoot(): string {
  const raw = process.env.COLLECTOR_STORAGE_STATE_DIR?.trim() || '';
  if (raw) {
    return path.isAbsolute(raw) ? raw : path.resolve(COLLECTOR_PACKAGE_ROOT, raw);
  }
  return path.join(COLLECTOR_PACKAGE_ROOT, 'data', 'storage-states');
}

export function ensureBrowserDataDirs(): void {
  for (const dir of [
    getBrowserProfileRoot(),
    get1688UserDataDir(),
    getPinduoduoUserDataDir(),
    getTaobaoTmallUserDataDir(),
    getStorageStateRoot(),
  ]) {
    fs.mkdirSync(dir, { recursive: true });
  }
}
