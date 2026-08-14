import { describe, expect, it } from "vitest";
import {
  assertPublicHttpUrl,
  assertPublicWebSocketUrl,
  isBlockedNetworkAddress,
} from "../public-url.js";

describe("collector public URL guard", () => {
  it("rejects private, loopback, metadata, and documentation addresses", async () => {
    for (const address of [
      "127.0.0.1",
      "10.0.0.1",
      "169.254.169.254",
      "192.168.1.1",
      "::1",
      "fc00::1",
      "fec0::1",
      "2001:db8::1",
      "::ffff:7f00:1",
      "64:ff9b::a9fe:a9fe",
      "2002:0a00:0001::1",
    ]) {
      expect(isBlockedNetworkAddress(address)).toBe(true);
      await expect(
        assertPublicHttpUrl(
          `http://${address.includes(":") ? `[${address}]` : address}/product`,
        ),
      ).rejects.toThrow("UNSAFE_TARGET_URL");
    }
  });

  it("applies the same public-address policy to WebSocket targets", async () => {
    const publicLookup = async () => [{ address: "93.184.216.34", family: 4 }];
    await expect(
      assertPublicWebSocketUrl("wss://example.com/socket", publicLookup),
    ).resolves.toBeInstanceOf(URL);
    await expect(
      assertPublicWebSocketUrl("ws://127.0.0.1/socket"),
    ).rejects.toThrow("private_address");
  });

  it("allows a public host only when every resolved address is public", async () => {
    const publicLookup = async () => [{ address: "93.184.216.34", family: 4 }];
    await expect(
      assertPublicHttpUrl("https://example.com/item", publicLookup),
    ).resolves.toBeInstanceOf(URL);

    const mixedLookup = async () => [
      { address: "93.184.216.34", family: 4 },
      { address: "10.0.0.2", family: 4 },
    ];
    await expect(
      assertPublicHttpUrl("https://example.com/item", mixedLookup),
    ).rejects.toThrow("private_address");
  });

  it("rejects credentials and non-HTTP schemes", async () => {
    await expect(assertPublicHttpUrl("file:///etc/passwd")).rejects.toThrow(
      "http_or_https_required",
    );
    await expect(
      assertPublicHttpUrl("https://user:pass@example.com/item"),
    ).rejects.toThrow("credentials_not_allowed");
  });
});
