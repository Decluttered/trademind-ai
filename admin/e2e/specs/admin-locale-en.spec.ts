import { test, expect } from '../fixtures/admin.fixture';
import { AUTH_TOKEN_KEY } from '../../src/constants/auth';
import { ADMIN_LOCALE_STORAGE_KEY } from '../../src/locale/localeMode';
import { seedAdminLocale } from '../utils/routes';

test.describe('@smoke Admin English locale', () => {
  test('login tabs and shops title render in English', async ({ admin, page }) => {
    await page.addInitScript(
      ([authKey, localeKey]) => {
        window.localStorage.removeItem(authKey);
        window.localStorage.setItem(localeKey, 'en');
      },
      [AUTH_TOKEN_KEY, ADMIN_LOCALE_STORAGE_KEY],
    );
    await page.goto('/user/login');
    await expect(page.getByRole('tab', { name: 'Sign in' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Register' })).toBeVisible();
    await expect(page.getByLabel('Language')).toBeVisible();
  });

  test('shops page chrome follows English locale', async ({ admin, page }) => {
    await seedAdminLocale(page, 'en');
    await admin.goto('/shops/manage');
    await expect(page.getByText('Shop management').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.getByLabel('Language')).toBeVisible();
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });
});
