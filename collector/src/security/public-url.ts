import { lookup } from "node:dns/promises";
import { isIP } from "node:net";
import type { BrowserContext } from "playwright";

type LookupAddress = { address: string; family: number };
type LookupAll = (hostname: string) => Promise<LookupAddress[]>;

const defaultLookup: LookupAll = async (hostname) =>
  lookup(hostname, { all: true, verbatim: true });

function isBlockedIPv4(address: string): boolean {
  const parts = address.split(".").map((part) => Number(part));
  if (
    parts.length !== 4 ||
    parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)
  ) {
    return true;
  }
  const [a, b, c] = parts;
  return (
    a === 0 ||
    a === 10 ||
    a === 127 ||
    (a === 100 && b >= 64 && b <= 127) ||
    (a === 169 && b === 254) ||
    (a === 172 && b >= 16 && b <= 31) ||
    (a === 192 && b === 0) ||
    (a === 192 && b === 168) ||
    (a === 192 && b === 0 && c === 2) ||
    (a === 198 && (b === 18 || b === 19)) ||
    (a === 198 && b === 51 && c === 100) ||
    (a === 203 && b === 0 && c === 113) ||
    a >= 224
  );
}

function isBlockedIPv6(rawAddress: string): boolean {
  const words = parseIPv6Words(rawAddress);
  if (!words) return true;
  const allZero = words.every((word) => word === 0);
  const loopback =
    words.slice(0, 7).every((word) => word === 0) && words[7] === 1;
  if (allZero || loopback) return true;
  if ((words[0] & 0xfe00) === 0xfc00) return true;
  if ((words[0] & 0xffc0) === 0xfe80 || (words[0] & 0xffc0) === 0xfec0)
    return true;
  if ((words[0] & 0xff00) === 0xff00) return true;
  if (words[0] === 0x2001 && words[1] === 0x0db8) return true;

  const embeddedIPv4 = `${words[6] >> 8}.${words[6] & 0xff}.${words[7] >> 8}.${words[7] & 0xff}`;
  const ipv4Compatible = words.slice(0, 6).every((word) => word === 0);
  const ipv4Mapped =
    words.slice(0, 5).every((word) => word === 0) && words[5] === 0xffff;
  const ipv4Translated =
    words.slice(0, 4).every((word) => word === 0) &&
    words[4] === 0xffff &&
    words[5] === 0;
  const nat64WellKnown =
    words[0] === 0x0064 &&
    words[1] === 0xff9b &&
    words.slice(2, 6).every((word) => word === 0);
  const sixToFour = words[0] === 0x2002;
  if (ipv4Compatible || ipv4Mapped || ipv4Translated || nat64WellKnown) {
    return isBlockedIPv4(embeddedIPv4);
  }
  if (sixToFour) {
    const sixToFourIPv4 = `${words[1] >> 8}.${words[1] & 0xff}.${words[2] >> 8}.${words[2] & 0xff}`;
    return isBlockedIPv4(sixToFourIPv4);
  }
  return false;
}

function parseIPv6Words(rawAddress: string): number[] | null {
  let address = rawAddress.toLowerCase().split("%")[0] ?? "";
  if (!address) return null;
  if (address.includes(".")) {
    const separator = address.lastIndexOf(":");
    if (separator < 0) return null;
    const ipv4 = address.slice(separator + 1);
    const parts = ipv4.split(".").map((part) => Number(part));
    if (
      parts.length !== 4 ||
      parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)
    ) {
      return null;
    }
    address = `${address.slice(0, separator)}:${((parts[0] << 8) | parts[1]).toString(16)}:${((parts[2] << 8) | parts[3]).toString(16)}`;
  }
  const halves = address.split("::");
  if (halves.length > 2) return null;
  const left = halves[0] ? halves[0].split(":") : [];
  const right = halves.length === 2 && halves[1] ? halves[1].split(":") : [];
  if (halves.length === 1 && left.length !== 8) return null;
  if (halves.length === 2 && left.length + right.length >= 8) return null;
  const fill =
    halves.length === 2 ? Array(8 - left.length - right.length).fill("0") : [];
  const rawWords = [...left, ...fill, ...right];
  if (
    rawWords.length !== 8 ||
    rawWords.some((word) => !/^[0-9a-f]{1,4}$/.test(word))
  )
    return null;
  return rawWords.map((word) => Number.parseInt(word, 16));
}

export function isBlockedNetworkAddress(address: string): boolean {
  const family = isIP(address);
  if (family === 4) return isBlockedIPv4(address);
  if (family === 6) return isBlockedIPv6(address);
  return true;
}

export async function assertPublicHttpUrl(
  rawUrl: string,
  resolveAll: LookupAll = defaultLookup,
): Promise<URL> {
  let parsed: URL;
  try {
    parsed = new URL(rawUrl);
  } catch {
    throw new Error("UNSAFE_TARGET_URL:invalid_url");
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("UNSAFE_TARGET_URL:http_or_https_required");
  }
  if (parsed.username || parsed.password) {
    throw new Error("UNSAFE_TARGET_URL:credentials_not_allowed");
  }
  const hostname = parsed.hostname
    .replace(/^\[|\]$/g, "")
    .replace(/\.$/, "")
    .toLowerCase();
  if (
    !hostname ||
    hostname === "localhost" ||
    hostname.endsWith(".localhost")
  ) {
    throw new Error("UNSAFE_TARGET_URL:private_host");
  }
  if (isIP(hostname)) {
    if (isBlockedNetworkAddress(hostname))
      throw new Error("UNSAFE_TARGET_URL:private_address");
    return parsed;
  }
  let addresses: LookupAddress[];
  try {
    addresses = await resolveAll(hostname);
  } catch {
    throw new Error("UNSAFE_TARGET_URL:dns_lookup_failed");
  }
  if (
    addresses.length === 0 ||
    addresses.some(({ address }) => isBlockedNetworkAddress(address))
  ) {
    throw new Error("UNSAFE_TARGET_URL:private_address");
  }
  return parsed;
}

export async function assertPublicWebSocketUrl(
  rawUrl: string,
  resolveAll: LookupAll = defaultLookup,
): Promise<URL> {
  let parsed: URL;
  try {
    parsed = new URL(rawUrl);
  } catch {
    throw new Error("UNSAFE_TARGET_URL:invalid_url");
  }
  if (parsed.protocol !== "ws:" && parsed.protocol !== "wss:") {
    throw new Error("UNSAFE_TARGET_URL:websocket_required");
  }
  const httpEquivalent = new URL(parsed.toString());
  httpEquivalent.protocol = parsed.protocol === "wss:" ? "https:" : "http:";
  await assertPublicHttpUrl(httpEquivalent.toString(), resolveAll);
  return parsed;
}

export async function installPublicNetworkGuard(
  context: BrowserContext,
): Promise<void> {
  await context.route("**/*", async (route) => {
    const requestUrl = route.request().url();
    let parsed: URL;
    try {
      parsed = new URL(requestUrl);
    } catch {
      await route.abort("blockedbyclient");
      return;
    }
    if (
      parsed.protocol === "data:" ||
      parsed.protocol === "blob:" ||
      parsed.protocol === "about:"
    ) {
      await route.continue();
      return;
    }
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      await route.abort("blockedbyclient");
      return;
    }
    try {
      await assertPublicHttpUrl(requestUrl);
      await route.continue();
    } catch {
      await route.abort("blockedbyclient");
    }
  });
  await context.routeWebSocket(/.*/, async (webSocket) => {
    try {
      await assertPublicWebSocketUrl(webSocket.url());
      webSocket.connectToServer();
    } catch {
      await webSocket.close({ code: 1008, reason: "blocked target" });
    }
  });
}
