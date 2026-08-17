import enUS from 'antd/locale/en_US';
import zhCN from 'antd/locale/zh_CN';
import deDE from 'antd/locale/de_DE';
import type { Locale } from 'antd/es/locale';
import type { AdminLocale } from './localeMode';

export function antdLocaleFor(locale: AdminLocale): Locale {
  if (locale === 'zh') return zhCN;
  if (locale === 'de') return deDE;
  return enUS;
}
