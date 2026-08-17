import { test, expect } from '../fixtures/admin.fixture';
import { ok } from '../mocks/envelope';
import { mindBayDraft } from '../mocks/mindbay';
import { expectNoRootOverflow } from '../utils/assertions';
import { seedAdminLocale } from '../utils/routes';

const viewports = [
  { width: 375, height: 812 },
  { width: 768, height: 1024 },
  { width: 1024, height: 768 },
  { width: 1280, height: 900 },
  { width: 1440, height: 960 },
];

test.describe('@mindbay Phase 1 fixture UI', () => {
  for (const viewport of viewports) {
    test(`products render without overflow at ${viewport.width}px`, async ({ admin, page }) => {
      test.setTimeout(90_000);
      await seedAdminLocale(page, 'de');
      await page.setViewportSize(viewport);
      await admin.goto('/mindbay/products');
      await expect(page.getByText('MindBay Fixture Produkt')).toBeVisible({ timeout: 60_000 });
      await expect(page.getByText('29,99 €')).toBeVisible();
      await expectNoRootOverflow(page);
    });
  }

  test('listing create cancel writes zero and confirm writes once', async ({ admin, page }) => {
    await seedAdminLocale(page, 'de');
    admin.writeGuard.allow({
      operation: 'create-mindbay-draft',
      method: 'POST',
      path: /^\/v1\/listing-drafts$/,
      response: ok(mindBayDraft),
    });
    await admin.goto('/mindbay/listing-studio');
    await expect(page.getByText('Haus & Garten')).toBeVisible({ timeout: 30000 });
    await page.getByRole('button', { name: 'Entwurf anlegen' }).click();
    await page.getByRole('button', { name: 'Abbrechen' }).click();
    await admin.writeGuard.expectRequestCount('create-mindbay-draft', 0);
    await page.getByRole('button', { name: 'Entwurf anlegen' }).click();
    await page.getByLabel('Source Product ID').fill('mindbay-product-e2e');
    await page.getByLabel('Kategorie').fill('Haus & Garten');
    await page.getByLabel('Preis in EUR').fill('49,90');
    await page.getByRole('button', { name: 'Anlegen', exact: true }).click();
    await admin.writeGuard.expectRequestCount('create-mindbay-draft', 1);
    expect(admin.writeGuard.calls('create-mindbay-draft')[0].postDataJSON).toEqual({
      sourceProductId: 'mindbay-product-e2e',
      category: 'Haus & Garten',
      priceCents: 4990,
      requiredSpecifics: [],
      specifics: {},
      imageAssetIds: [],
    });
    expect(admin.writeGuard.calls('create-mindbay-draft')[0].headers['idempotency-key']).toBeTruthy();
  });
});
