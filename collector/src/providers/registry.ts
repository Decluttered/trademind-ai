import type { CollectorProvider } from './collector-provider.js';
import type { CollectProviderPublic } from '../types/provider-meta.js';
import { amazonDECollectorProvider } from './sourceAmazon/index.js';

const providers: CollectorProvider[] = [amazonDECollectorProvider];

const bySource = new Map<string, CollectorProvider>(
  providers.map((p) => [p.sourceId.toLowerCase(), p]),
);
bySource.set('amazon', amazonDECollectorProvider);

export function getProviderBySource(source: string): CollectorProvider | undefined {
  return bySource.get(source.trim().toLowerCase());
}

export function listRegisteredSources(): string[] {
  return providers.map((p) => p.sourceId);
}

/** Public-facing list: stable order for easy display in the admin panel */
export function listProviderPublicMetas(): CollectProviderPublic[] {
  return providers.map((p) => ({
    source: p.sourceId,
    ...p.meta,
  }));
}
