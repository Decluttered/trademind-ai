import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { useAntdConfigSetter } from '@umijs/max';
import type { AdminLocale } from './localeMode';
import {
  applyAdminLocale,
  getStoredAdminLocale,
  persistAdminLocale,
} from './localeMode';
import { translate, type TranslateOptions } from './translate';
import { antdLocaleFor } from './antdLocale';

type LocaleContextValue = {
  locale: AdminLocale;
  setLocale: (locale: AdminLocale) => void;
  t: (key: string, options?: TranslateOptions) => string;
};

const LocaleContext = createContext<LocaleContextValue | null>(null);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const setAntdConfig = useAntdConfigSetter();
  const [locale, setLocaleState] = useState<AdminLocale>(getStoredAdminLocale);

  useEffect(() => {
    applyAdminLocale(locale);
  }, [locale]);

  const setLocale = useCallback(
    (next: AdminLocale) => {
      setLocaleState(next);
      persistAdminLocale(next);
      setAntdConfig({ locale: antdLocaleFor(next) });
    },
    [setAntdConfig],
  );

  const t = useCallback(
    (key: string, options?: TranslateOptions) => translate(locale, key, options),
    [locale],
  );

  const value = useMemo(
    () => ({ locale, setLocale, t }),
    [locale, setLocale, t],
  );

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  );
}

export function useLocale(): LocaleContextValue {
  const ctx = useContext(LocaleContext);
  if (!ctx) {
    const locale = getStoredAdminLocale();
    return {
      locale,
      setLocale: (next) => {
        persistAdminLocale(next);
      },
      t: (key, options) => translate(locale, key, options),
    };
  }
  return ctx;
}
