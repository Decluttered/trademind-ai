import type { CollectProviderRow, CollectProviderStatus } from '@/services/collectProviders';

/** URL query `provider` values for /settings/collector */
export type CollectSettingsProviderKey = 'amazon.de';

export type CollectSettingsProviderOption = {
  key: CollectSettingsProviderKey;
  label: string;
  /** Collect hub / task `source` id */
  source: string;
  planned?: boolean;
};

export const COLLECT_SETTINGS_PROVIDER_OPTIONS: CollectSettingsProviderOption[] = [
  { key: 'amazon.de', label: 'Amazon.de 采集器', source: 'amazon.de' },
];

const PROVIDER_KEY_SET = new Set<string>(COLLECT_SETTINGS_PROVIDER_OPTIONS.map((o) => o.key));

/** Map collect hub card `source` → settings URL `provider`. */
export function collectSourceToSettingsProvider(source: string): CollectSettingsProviderKey {
  const s = source.trim().toLowerCase();
  if (s === 'amazon' || s === 'amazon.de') {
    return 'amazon.de';
  }
  return 'amazon.de';
}

export function resolveCollectSettingsProvider(raw: string | null | undefined): CollectSettingsProviderKey {
  const key = (raw ?? '').trim();
  if (PROVIDER_KEY_SET.has(key)) {
    return key as CollectSettingsProviderKey;
  }
  return 'amazon.de';
}

export function collectSettingsPath(source: string): string {
  const provider = collectSourceToSettingsProvider(source);
  return `/settings/collector?provider=${encodeURIComponent(provider)}`;
}

export function findCollectSettingsOption(key: CollectSettingsProviderKey): CollectSettingsProviderOption {
  return COLLECT_SETTINGS_PROVIDER_OPTIONS.find((o) => o.key === key) ?? COLLECT_SETTINGS_PROVIDER_OPTIONS[0];
}

export function providerStatusFromRows(
  rows: CollectProviderRow[],
  source: string,
): CollectProviderStatus | undefined {
  const key = source.trim().toLowerCase();
  return rows.find((r) => r.source.trim().toLowerCase() === key)?.status;
}

export function isPlannedCollectProvider(
  rows: CollectProviderRow[],
  option: CollectSettingsProviderOption,
): boolean {
  const status = providerStatusFromRows(rows, option.source);
  if (status) return status === 'planned';
  return !!option.planned;
}

export function collectSettingsConfigButtonLabel(status?: CollectProviderStatus): string {
  return status === 'planned' ? '查看配置' : '采集设置';
}
