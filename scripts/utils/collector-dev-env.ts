import { randomBytes } from "node:crypto";

export type CollectorDevToken = {
  token: string;
  generated: boolean;
};

/** Resolve the one Token shared by local backend and Collector child processes. */
export function resolveCollectorDevToken(
  processToken: string | undefined,
  configToken: string | undefined,
  generateToken: () => string = () => randomBytes(32).toString("hex"),
): CollectorDevToken {
  const inherited = processToken?.trim();
  if (inherited) return { token: inherited, generated: false };

  const configured = configToken?.trim();
  if (configured) return { token: configured, generated: false };

  return { token: generateToken(), generated: true };
}
