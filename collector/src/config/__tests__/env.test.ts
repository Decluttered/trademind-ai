import { afterEach, describe, expect, it } from "vitest";
import {
  getBrowserHeadless,
  getCollectorServiceToken,
  getDefaultNavigationTimeoutMs,
  getHeadersTimeoutMs,
  getHttpPort,
  getMaxRequestBodyBytes,
  getRequestTimeoutMs,
} from "../env.js";

const KEYS = [
  "COLLECTOR_HTTP_ADDR",
  "COLLECTOR_GOTO_TIMEOUT_MS",
  "COLLECTOR_HEADLESS",
  "COLLECTOR_SERVICE_TOKEN",
  "COLLECTOR_MAX_REQUEST_BODY_BYTES",
  "COLLECTOR_REQUEST_TIMEOUT_MS",
  "COLLECTOR_HEADERS_TIMEOUT_MS",
] as const;
const original = Object.fromEntries(KEYS.map((key) => [key, process.env[key]]));

afterEach(() => {
  for (const key of KEYS) {
    const value = original[key];
    if (value === undefined) delete process.env[key];
    else process.env[key] = value;
  }
});

describe("collector env helpers", () => {
  it("parses colon-prefixed HTTP ports and falls back safely", () => {
    process.env.COLLECTOR_HTTP_ADDR = ":3201";
    expect(getHttpPort()).toBe(3201);

    process.env.COLLECTOR_HTTP_ADDR = "not-a-port";
    expect(getHttpPort()).toBe(3100);
  });

  it("uses a positive navigation timeout", () => {
    process.env.COLLECTOR_GOTO_TIMEOUT_MS = "60000";
    expect(getDefaultNavigationTimeoutMs()).toBe(60000);

    process.env.COLLECTOR_GOTO_TIMEOUT_MS = "-1";
    expect(getDefaultNavigationTimeoutMs()).toBe(45000);
  });

  it("keeps headless mode on unless explicitly disabled", () => {
    delete process.env.COLLECTOR_HEADLESS;
    expect(getBrowserHeadless()).toBe(true);

    process.env.COLLECTOR_HEADLESS = "0";
    expect(getBrowserHeadless()).toBe(false);

    process.env.COLLECTOR_HEADLESS = "false";
    expect(getBrowserHeadless()).toBe(false);
  });

  it("reads the internal service token without inventing a default", () => {
    delete process.env.COLLECTOR_SERVICE_TOKEN;
    expect(getCollectorServiceToken()).toBe("");
    process.env.COLLECTOR_SERVICE_TOKEN = "  test-collector-token  ";
    expect(getCollectorServiceToken()).toBe("test-collector-token");
  });

  it("bounds HTTP request limits and timeouts", () => {
    process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES = "2048";
    process.env.COLLECTOR_REQUEST_TIMEOUT_MS = "20000";
    process.env.COLLECTOR_HEADERS_TIMEOUT_MS = "12000";
    expect(getMaxRequestBodyBytes()).toBe(2048);
    expect(getRequestTimeoutMs()).toBe(20000);
    expect(getHeadersTimeoutMs()).toBe(12000);

    process.env.COLLECTOR_MAX_REQUEST_BODY_BYTES = "999999999";
    process.env.COLLECTOR_REQUEST_TIMEOUT_MS = "-1";
    process.env.COLLECTOR_HEADERS_TIMEOUT_MS = "0";
    expect(getMaxRequestBodyBytes()).toBe(1048576);
    expect(getRequestTimeoutMs()).toBe(15000);
    expect(getHeadersTimeoutMs()).toBe(10000);
  });
});
