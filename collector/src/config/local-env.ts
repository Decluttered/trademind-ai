import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseEnv } from "node:util";

const collectorRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
);
const repoRoot = path.resolve(collectorRoot, "..");

function isCollectorRuntimeKey(key: string): boolean {
  return key.startsWith("COLLECTOR_") || key === "BROWSER_PROFILE_ROOT";
}

/**
 * Apply only Collector-owned values from a dotenv file without overriding
 * values explicitly supplied by the process environment.
 */
export function loadCollectorEnvFile(
  envPath: string,
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const contents = fs.readFileSync(envPath, "utf8").replace(/^\uFEFF/u, "");
  const values = parseEnv(contents);
  for (const [key, value] of Object.entries(values)) {
    if (isCollectorRuntimeKey(key) && environment[key] === undefined) {
      environment[key] = value;
    }
  }
}

/** Load root .env, falling back to backend/.env to match the backend startup path. */
export function loadCollectorDevelopmentEnv(
  environment: NodeJS.ProcessEnv = process.env,
): void {
  const rootEnv = path.join(repoRoot, ".env");
  const backendEnv = path.join(repoRoot, "backend", ".env");
  const envPath = fs.existsSync(rootEnv) ? rootEnv : backendEnv;
  if (!fs.existsSync(envPath)) return;

  try {
    loadCollectorEnvFile(envPath, environment);
  } catch (error) {
    throw new Error(
      `collector failed to load local environment from ${envPath}`,
      { cause: error },
    );
  }
}
