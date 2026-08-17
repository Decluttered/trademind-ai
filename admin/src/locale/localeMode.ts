export const ADMIN_LOCALE_STORAGE_KEY = 'trademind_admin_locale';

export type AdminLocale = 'en' | 'zh' | 'de';

export const ADMIN_LOCALES: AdminLocale[] = ['en', 'zh', 'de'];

export const DEFAULT_ADMIN_LOCALE: AdminLocale = 'en';

type LocaleStorage = Pick<Storage, 'getItem' | 'setItem'>;

export function isAdminLocale(value: unknown): value is AdminLocale {
  return value === 'en' || value === 'zh' || value === 'de';
}

export function readAdminLocale(
  storage?: Pick<LocaleStorage, 'getItem'>,
): AdminLocale {
  try {
    const raw = storage?.getItem(ADMIN_LOCALE_STORAGE_KEY);
    return isAdminLocale(raw) ? raw : DEFAULT_ADMIN_LOCALE;
  } catch {
    return DEFAULT_ADMIN_LOCALE;
  }
}

export function getStoredAdminLocale(): AdminLocale {
  return readAdminLocale(
    typeof window === 'undefined' ? undefined : window.localStorage,
  );
}

export function applyAdminLocale(locale: AdminLocale) {
  if (typeof document === 'undefined') return;
  document.documentElement.lang =
    locale === 'zh' ? 'zh-CN' : locale === 'de' ? 'de' : 'en';
}

export function persistAdminLocale(
  locale: AdminLocale,
  storage?: Pick<LocaleStorage, 'setItem'>,
) {
  const targetStorage =
    storage ??
    (typeof window === 'undefined' ? undefined : window.localStorage);
  try {
    targetStorage?.setItem(ADMIN_LOCALE_STORAGE_KEY, locale);
  } catch {
    // Active locale still applies when browser storage is unavailable.
  }
  applyAdminLocale(locale);
}
