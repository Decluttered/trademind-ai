import { mkdtemp, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { loadCollectorEnvFile } from "../local-env.js";

const temporaryDirectories: string[] = [];

afterEach(async () => {
  await Promise.all(
    temporaryDirectories
      .splice(0)
      .map((dir) => rm(dir, { recursive: true, force: true })),
  );
});

describe("loadCollectorEnvFile", () => {
  it("loads only Collector configuration and preserves explicit process values", async () => {
    const dir = await mkdtemp(
      path.join(os.tmpdir(), "trademind-collector-env-"),
    );
    temporaryDirectories.push(dir);
    const envPath = path.join(dir, ".env");
    await writeFile(
      envPath,
      [
        "COLLECTOR_SERVICE_TOKEN=file-token",
        "COLLECTOR_HTTP_ADDR=:3210",
        "BROWSER_PROFILE_ROOT=profiles",
        "DB_PASSWORD=must-not-be-imported",
      ].join("\n"),
      "utf8",
    );

    const environment: NodeJS.ProcessEnv = {
      COLLECTOR_SERVICE_TOKEN: "process-token",
    };
    loadCollectorEnvFile(envPath, environment);

    expect(environment).toMatchObject({
      COLLECTOR_SERVICE_TOKEN: "process-token",
      COLLECTOR_HTTP_ADDR: ":3210",
      BROWSER_PROFILE_ROOT: "profiles",
    });
    expect(environment.DB_PASSWORD).toBeUndefined();
  });
});
