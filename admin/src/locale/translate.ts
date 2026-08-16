import type { AdminLocale } from './localeMode';
import { deMessages } from './messages/de';
import { enMessages } from './messages/en';
import { zhMessages } from './messages/zh';

export type MessageTree = { [key: string]: string | MessageTree };

const CATALOGS: Record<AdminLocale, MessageTree> = {
  en: enMessages,
  zh: zhMessages,
  de: deMessages,
};

function lookup(tree: MessageTree, path: string): string | undefined {
  const parts = path.split('.');
  let cursor: string | MessageTree | undefined = tree;
  for (const part of parts) {
    if (!cursor || typeof cursor === 'string') return undefined;
    cursor = cursor[part];
  }
  return typeof cursor === 'string' ? cursor : undefined;
}

export type TranslateOptions = {
  values?: Record<string, string | number>;
};

function interpolate(template: string, values?: Record<string, string | number>) {
  if (!values) return template;
  return template.replace(/\{\{(\w+)\}\}/g, (_, key: string) =>
    values[key] === undefined ? `{{${key}}}` : String(values[key]),
  );
}

/**
 * Resolve a message key with fallback: current → en → zh → key.
 */
export function translate(
  locale: AdminLocale,
  key: string,
  options?: TranslateOptions,
): string {
  const resolved =
    lookup(CATALOGS[locale], key) ??
    (locale !== 'en' ? lookup(CATALOGS.en, key) : undefined) ??
    (locale !== 'zh' ? lookup(CATALOGS.zh, key) : undefined) ??
    key;
  return interpolate(resolved, options?.values);
}
