export {
  ADMIN_LOCALE_STORAGE_KEY,
  ADMIN_LOCALES,
  DEFAULT_ADMIN_LOCALE,
  applyAdminLocale,
  getStoredAdminLocale,
  isAdminLocale,
  persistAdminLocale,
  readAdminLocale,
} from './localeMode';
export type { AdminLocale } from './localeMode';
export { LocaleProvider, useLocale } from './LocaleProvider';
export { translate } from './translate';
export type { TranslateOptions, MessageTree } from './translate';
export { antdLocaleFor } from './antdLocale';
export { MENU_LOCALE_KEYS } from './menuLocaleKeys';
