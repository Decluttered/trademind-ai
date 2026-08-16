import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ADMIN_LOCALE_STORAGE_KEY,
  applyAdminLocale,
  persistAdminLocale,
  readAdminLocale,
} from '../localeMode';
import { translate } from '../translate';

beforeEach(() => {
  document.documentElement.removeAttribute('lang');
});

describe('admin locale mode', () => {
  it('defaults to English when there is no valid stored preference', () => {
    expect(readAdminLocale({ getItem: () => null })).toBe('en');
    expect(readAdminLocale({ getItem: () => 'fr' })).toBe('en');
    expect(
      readAdminLocale({
        getItem: () => {
          throw new Error('storage unavailable');
        },
      }),
    ).toBe('en');
  });

  it('persists locale and updates document lang', () => {
    const setItem = vi.fn();
    persistAdminLocale('zh', { setItem });
    expect(setItem).toHaveBeenCalledWith(ADMIN_LOCALE_STORAGE_KEY, 'zh');
    expect(document.documentElement.lang).toBe('zh-CN');

    applyAdminLocale('de');
    expect(document.documentElement.lang).toBe('de');

    applyAdminLocale('en');
    expect(document.documentElement.lang).toBe('en');
  });
});

describe('translate fallback', () => {
  it('falls back from de missing keys to en then zh', () => {
    expect(translate('de', 'login.tabLogin')).toBe('Anmelden');
    expect(translate('de', 'login.successLogin')).toBe('Signed in');
    expect(translate('en', 'page.shops.title')).toBe('Shop management');
    expect(translate('zh', 'page.shops.title')).toBe('店铺管理');
  });

  it('interpolates values', () => {
    expect(
      translate('en', 'page.users.assignShops', { values: { name: 'Ada' } }),
    ).toBe('Assign shop permissions — Ada');
  });
});
