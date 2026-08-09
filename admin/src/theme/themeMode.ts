export const THEME_MODE_STORAGE_KEY = 'trademind_admin_theme_mode';

export type ThemeMode = 'light' | 'dark';

type ThemeStorage = Pick<Storage, 'getItem' | 'setItem'>;

export function readThemeMode(
  storage?: Pick<ThemeStorage, 'getItem'>,
): ThemeMode {
  try {
    return storage?.getItem(THEME_MODE_STORAGE_KEY) === 'dark'
      ? 'dark'
      : 'light';
  } catch {
    return 'light';
  }
}

export function getStoredThemeMode(): ThemeMode {
  return readThemeMode(
    typeof window === 'undefined' ? undefined : window.localStorage,
  );
}

export function applyThemeMode(mode: ThemeMode) {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.theme = mode;
  document.documentElement.style.colorScheme = mode;
}

export function persistThemeMode(
  mode: ThemeMode,
  storage?: Pick<ThemeStorage, 'setItem'>,
) {
  const targetStorage =
    storage ??
    (typeof window === 'undefined' ? undefined : window.localStorage);
  try {
    targetStorage?.setItem(THEME_MODE_STORAGE_KEY, mode);
  } catch {
    // The active theme still changes when browser storage is unavailable.
  }
  applyThemeMode(mode);
}
