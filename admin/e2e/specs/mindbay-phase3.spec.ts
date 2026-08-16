import { test, expect } from '../fixtures/admin.fixture';
import { ok } from '../mocks/envelope';
import { mindBayDecision } from '../mocks/mindbay';
import { expectNoRootOverflow } from '../utils/assertions';

const viewports = [{ width: 1440, height: 900 }, { width: 1280, height: 800 }, { width: 1024, height: 768 }, { width: 768, height: 900 }, { width: 375, height: 812 }];

test.describe('@mindbay Phase 3 monitoring and profit', () => {
  for (const viewport of viewports) {
    test(`monitoring has no root overflow at ${viewport.width}x${viewport.height}`, async ({ admin, page }) => {
      await page.setViewportSize(viewport);
      await admin.goto('/mindbay/monitoring');
      await expect(page.getByText('DRY_RUN bleibt der sichere Standard')).toBeVisible({ timeout: 30_000 });
      await expectNoRootOverflow(page);
      await admin.writeGuard.expectRequestCount('unexpected', 0);
    });
  }

  test('monitor run and apply require separate confirmations', async ({ admin, page }) => {
    admin.writeGuard.allow({ operation: 'monitor-run', method: 'POST', path: /^\/v1\/monitor-runs$/, response: ok({ run: { id: 'monitor-run-e2e', status: 'SUCCEEDED' }, decision: mindBayDecision }) });
    admin.writeGuard.allow({ operation: 'decision-apply', method: 'POST', path: /^\/v1\/price-decisions\/price-decision-e2e\/apply$/, response: ok(mindBayDecision) });
    await admin.goto('/mindbay/monitoring');
    await page.getByRole('combobox', { name: 'eBay Listing' }).click();
    await page.getByText(/MB-E2E-1/).click();
    await page.getByRole('combobox', { name: 'Preisregel' }).click();
    await page.getByText(/E2E Guardrail/).click();
    await page.getByRole('button', { name: 'Monitoring ausführen' }).click();
    await page.getByRole('dialog').getByRole('button', { name: /Abbrechen|取\s*消/ }).click();
    await admin.writeGuard.expectRequestCount('monitor-run', 0);
    await page.getByRole('button', { name: 'Monitoring ausführen' }).click();
    await page.getByRole('button', { name: 'Lauf starten' }).click();
    await admin.writeGuard.expectRequestCount('monitor-run', 1);
    expect(admin.writeGuard.calls('monitor-run')[0].postDataJSON).toEqual({ marketplaceListingId: 'marketplace-listing-e2e', priceRuleId: 'price-rule-e2e', trigger: 'manual' });
    await page.getByRole('button', { name: 'Anwenden', exact: true }).click();
    await page.getByRole('dialog').getByRole('button', { name: /Abbrechen|取\s*消/ }).click();
    await admin.writeGuard.expectRequestCount('decision-apply', 0);
    await page.getByRole('button', { name: 'Anwenden', exact: true }).click();
    await page.getByRole('button', { name: 'Preis anwenden' }).click();
    await admin.writeGuard.expectRequestCount('decision-apply', 1);
  });

  test('profit keeps forecast and realized states separate', async ({ admin, page }) => {
    await admin.goto('/mindbay/profit');
    await expect(page.getByText('Erwartete Marge').first()).toBeVisible();
    await expect(page.getByText('prognostiziert')).toBeVisible();
    await expect(page.getByText('Wird ab Phase 4 mit Sales gefüllt.')).toBeVisible();
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });
});
