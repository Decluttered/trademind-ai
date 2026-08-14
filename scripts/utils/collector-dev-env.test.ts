import { describe, expect, it } from "vitest";

import { resolveCollectorDevToken } from "./collector-dev-env.js";

describe("resolveCollectorDevToken", () => {
  it("prefers an inherited Token over the local config value", () => {
    expect(resolveCollectorDevToken(" process-token ", "config-token")).toEqual(
      {
        token: "process-token",
        generated: false,
      },
    );
  });

  it("uses the configured Token when no inherited Token is available", () => {
    expect(resolveCollectorDevToken(undefined, " config-token ")).toEqual({
      token: "config-token",
      generated: false,
    });
  });

  it("creates an in-memory Token only when neither source is configured", () => {
    expect(resolveCollectorDevToken("", "", () => "generated-token")).toEqual({
      token: "generated-token",
      generated: true,
    });
  });
});
