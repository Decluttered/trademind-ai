import { test, expect } from '../fixtures/admin.fixture';
import { ok } from '../mocks/envelope';
import { mindBayPreviewSlot } from '../mocks/mindbay';
import { E2E_EBAY_SHOP_ID } from '../mocks/publish';
import { expectNoRootOverflow } from '../utils/assertions';

const viewports = [
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 900 },
  { width: 375, height: 812 },
];

test.describe('@mindbay Phase 2 planner', () => {
  for (const viewport of viewports) {
    test(`planner is readable without overflow at ${viewport.width}x${viewport.height}`, async ({ admin, page }) => {
      await page.setViewportSize(viewport);
      await admin.goto('/mindbay/planner');
      await expect(page.getByText('Sicherer Standard: DRY_RUN + eBay Sandbox')).toBeVisible({ timeout: 30_000 });
      await expectNoRootOverflow(page);
      await admin.writeGuard.expectRequestCount('unexpected', 0);
    });
  }

  test('preview stays non-persistent and apply requires explicit confirmation', async ({ admin, page }) => {
    admin.writeGuard.allow({ operation: 'calendar-preview', method: 'POST', path: /^\/v1\/calendar\/preview$/, response: ok({ slots: [mindBayPreviewSlot], unplaced: 0 }) });
    admin.writeGuard.allow({ operation: 'calendar-apply', method: 'POST', path: /^\/v1\/calendar\/apply$/, response: ok({ slots: [], jobs: [] }) });
    await admin.goto('/mindbay/planner');
    await page.getByRole('button', { name: 'Preview berechnen' }).click();
    await expect(page.getByText('MindBay READY Fixture')).toBeVisible();
    await admin.writeGuard.expectRequestCount('calendar-preview', 1);
    await admin.writeGuard.expectRequestCount('calendar-apply', 0);

    await page.getByLabel('eBay Shop').click();
    await page.getByText('E2E eBay Sandbox · authorized').click();
    await page.getByLabel('Location Key').fill('de-warehouse');
    await page.getByLabel('Payment Policy').fill('payment-policy');
    await page.getByLabel('Return Policy').fill('return-policy');
    await page.getByLabel('Fulfillment Policy').fill('fulfillment-policy');
    await page.getByRole('button', { name: 'Slots einplanen' }).click();
    await page.getByRole('button', { name: 'Abbrechen' }).click();
    await admin.writeGuard.expectRequestCount('calendar-apply', 0);
    await page.getByRole('button', { name: 'Slots einplanen' }).click();
    await page.getByRole('button', { name: 'Einplanen', exact: true }).click();
    await admin.writeGuard.expectRequestCount('calendar-apply', 1);
    const call = admin.writeGuard.calls('calendar-apply')[0];
    expect(call.headers['idempotency-key']).toBeTruthy();
    expect(call.postDataJSON).toMatchObject({ shopId: E2E_EBAY_SHOP_ID, marketplace: 'EBAY_DE', slots: [mindBayPreviewSlot], publishConfig: { merchantLocationKey: 'de-warehouse', paymentPolicyId: 'payment-policy', returnPolicyId: 'return-policy', fulfillmentPolicyId: 'fulfillment-policy', condition: 'NEW', quantity: 1 } });
  });
});
