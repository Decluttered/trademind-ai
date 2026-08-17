/**
 * Environment variables (injected by docker / systemd / .env; secrets are not written into code defaults).
 */
import {
  getBrowserProfileRoot,
  get1688UserDataDir,
  getStorageStateRoot,
} from "../browser/browser-paths.js";

export { getBrowserProfileRoot, get1688UserDataDir, getStorageStateRoot };

export function getHttpPort(): number {
  const raw = process.env.COLLECTOR_HTTP_ADDR ?? ":3100";
  const n = Number(String(raw).replace(/^\:/, ""));
  return Number.isFinite(n) && n > 0 ? n : 3100;
}

export function getDefaultNavigationTimeoutMs(): number {
  const n = Number(process.env.COLLECTOR_GOTO_TIMEOUT_MS ?? "45000");
  return Number.isFinite(n) && n > 0 ? n : 45000;
}

export function getBrowserHeadless(): boolean {
  const v = process.env.COLLECTOR_HEADLESS;
  if (v === "0" || v === "false") return false;
  return true;
}

export function getCollectorServiceToken(): string {
  return String(process.env.COLLECTOR_SERVICE_TOKEN ?? "").trim();
}

export function getMaxRequestBodyBytes(): number {
  const n = Number(process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES ?? "1048576");
  return Number.isFinite(n) && n >= 1024 && n <= 16 * 1024 * 1024
    ? Math.floor(n)
    : 1048576;
}

export function getRequestTimeoutMs(): number {
  const n = Number(process.env.COLLECTOR_REQUEST_TIMEOUT_MS ?? "15000");
  return Number.isFinite(n) && n >= 1000 && n <= 120000 ? Math.floor(n) : 15000;
}

export function getHeadersTimeoutMs(): number {
  const n = Number(process.env.COLLECTOR_HEADERS_TIMEOUT_MS ?? "10000");
  return Number.isFinite(n) && n >= 1000 && n <= 60000 ? Math.floor(n) : 10000;
}

/** @deprecated use getBrowserProfileRoot() */
export function getBrowserProfileBaseDir(): string {
  return getBrowserProfileRoot();
}

/** @deprecated use getStorageStateRoot() */
export function getStorageStateBaseDir(): string {
  return getStorageStateRoot();
}
