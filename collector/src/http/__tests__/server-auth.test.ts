import { request } from "node:http";
import type { AddressInfo } from "node:net";
import { afterEach, describe, expect, it } from "vitest";
import type { BrowserManager } from "../../browser/manager.js";
import { createCollectorServer, serviceTokensEqual } from "../server.js";

const originalToken = process.env.COLLECTOR_SERVICE_TOKEN;
const originalBodyLimit = process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES;
const originalNodeEnv = process.env.NODE_ENV;

afterEach(() => {
  if (originalToken === undefined) delete process.env.COLLECTOR_SERVICE_TOKEN;
  else process.env.COLLECTOR_SERVICE_TOKEN = originalToken;
  if (originalBodyLimit === undefined)
    delete process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES;
  else process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES = originalBodyLimit;
  if (originalNodeEnv === undefined) delete process.env.NODE_ENV;
  else process.env.NODE_ENV = originalNodeEnv;
});

async function requestServer(
  path: string,
  options: {
    token?: string;
    body?: string;
    collectorToken?: string;
    bodyLimit?: number;
  } = {},
): Promise<{ status: number; body: { error?: { code?: string } } }> {
  if (options.collectorToken === undefined)
    delete process.env.COLLECTOR_SERVICE_TOKEN;
  else process.env.COLLECTOR_SERVICE_TOKEN = options.collectorToken;
  if (options.bodyLimit === undefined)
    delete process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES;
  else process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES = String(options.bodyLimit);
  const browser = {} as BrowserManager;
  const server = createCollectorServer(browser);
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  try {
    const address = server.address() as AddressInfo;
    return await new Promise((resolve, reject) => {
      const body = options.body ?? "";
      const req = request(
        {
          host: "127.0.0.1",
          port: address.port,
          path,
          method: body ? "POST" : "GET",
          headers: {
            ...(options.token
              ? { Authorization: `Bearer ${options.token}` }
              : {}),
            ...(body
              ? {
                  "Content-Type": "application/json",
                  "Content-Length": Buffer.byteLength(body),
                }
              : {}),
          },
        },
        (res) => {
          const chunks: Buffer[] = [];
          res.on("data", (chunk) => chunks.push(Buffer.from(chunk)));
          res.on("end", () =>
            resolve({
              status: res.statusCode ?? 0,
              body: JSON.parse(Buffer.concat(chunks).toString("utf8")) as {
                error?: { code?: string };
              },
            }),
          );
        },
      );
      req.on("error", reject);
      req.end(body);
    });
  } finally {
    await new Promise<void>((resolve, reject) =>
      server.close((error) => (error ? reject(error) : resolve())),
    );
  }
}

describe("collector service authentication", () => {
  it("compares service tokens using fixed-length digests", () => {
    expect(serviceTokensEqual("same-token", "same-token")).toBe(true);
    expect(serviceTokensEqual("wrong", "same-token")).toBe(false);
    expect(serviceTokensEqual("", "same-token")).toBe(false);
  });

  it("keeps health public and fails closed when service authentication is not configured", async () => {
    expect((await requestServer("/health")).status).toBe(200);
    const response = await requestServer("/v1/providers");
    expect(response.status).toBe(503);
    expect(response.body.error?.code).toBe("SERVICE_AUTH_NOT_CONFIGURED");
  });

  it("requires the configured bearer token on protected routes", async () => {
    const unauthorized = await requestServer("/v1/providers", {
      token: "wrong",
      collectorToken: "collector-test-token",
    });
    expect(unauthorized.status).toBe(401);
    const authorized = await requestServer("/v1/providers", {
      token: "collector-test-token",
      collectorToken: "collector-test-token",
    });
    expect(authorized.status).toBe(200);
  });

  it("rejects weak service authentication during production startup", () => {
    process.env.NODE_ENV = "production";
    process.env.COLLECTOR_SERVICE_TOKEN = "short-token";
    expect(() => createCollectorServer({} as BrowserManager)).toThrow(
      "COLLECTOR_CONFIG_INVALID",
    );

    process.env.COLLECTOR_SERVICE_TOKEN = "x".repeat(32);
    const server = createCollectorServer({} as BrowserManager);
    server.close();
  });

  it("returns 413 when a protected request exceeds the configured body limit", async () => {
    const response = await requestServer(
      "/v1/providers/pinduoduo/check-login",
      {
        token: "collector-test-token",
        collectorToken: "collector-test-token",
        bodyLimit: 1024,
        body: JSON.stringify({
          url: `https://example.com/${"x".repeat(2048)}`,
        }),
      },
    );
    expect(response.status).toBe(413);
    expect(response.body.error?.code).toBe("REQUEST_TOO_LARGE");
  });
});
