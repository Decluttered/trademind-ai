import { test, expect } from '../fixtures/admin.fixture';
import { ok } from '../mocks/envelope';
import { p9Run } from '../mocks/inventory-sync-p9';
import { expectNoRootOverflow } from '../utils/assertions';

const viewports = [
  { width: 375, height: 812 },
  { width: 768, height: 1024 },
  { width: 1024, height: 768 },
  { width: 1280, height: 900 },
  { width: 1440, height: 960 },
];

test.describe('@p9-inventory-sync @p9-batch7 Admin fixture binding center', () => {
  for (const viewport of viewports) {
    test(`dashboard renders without overflow at ${viewport.width}px`, async ({ admin, page }) => {
      await page.setViewportSize(viewport);
      await admin.goto('/ops/inventory-sync');
      await expect(page.getByText('productionReady=false')).toBeVisible({ timeout: 30000 });
      await expect(page.getByText('run-p9-e2e-1').first()).toBeVisible();
      await expect(page.getByText('success_single_page').first()).toBeVisible();
      await expectNoRootOverflow(page);
    });
  }

  test('create run sends only fixture write body through guarded endpoint', async ({ admin, page }) => {
    admin.writeGuard.allow({
      operation: 'create-run',
      method: 'POST',
      path: /^\/api\/v1\/inventory-sync\/runs$/,
      response: ok(p9Run),
    });

    await admin.goto('/ops/inventory-sync');
    await expect(page.getByText('productionReady=false')).toBeVisible({ timeout: 30000 });
    await page.getByTestId('p9-create-run').click();
    await page.getByTestId('p9-create-run-shop').fill('shop-p9-e2e');
    await page.getByTestId('p9-create-run-submit').click();

    await admin.writeGuard.expectRequestCount('create-run', 1);
    expect(admin.writeGuard.calls('create-run')[0].postDataJSON).toEqual({
      shopConnectionId: 'shop-p9-e2e',
      platform: 'douyin',
      providerMode: 'mock',
      fixtureScenario: 'success_single_page',
    });
    await expect(page).toHaveURL(/\/ops\/inventory-sync\/runs\/run-p9-e2e-1/);
  });

  test('calibration, manual binding, and binding history routes use fixture copy', async ({ admin, page }) => {
    await admin.goto('/ops/inventory-sync/calibration?snapshotId=snapshot-p9-e2e-1');
    await expect(page.getByText('productionReady=false')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText(/calibration-p9/).first()).toBeVisible();
    await expect(page.getByText('85%').first()).toBeVisible();
    await expectNoRootOverflow(page);

    await admin.goto('/ops/inventory-sync/manual-bindings?requestId=manual-p9-e2e-1');
    await expect(page.getByText('productionReady=false')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText(/manual-p9/).first()).toBeVisible();
    await expect(page.getByText('manual_review_required').first()).toBeVisible();
    await expectNoRootOverflow(page);

    await admin.goto('/ops/inventory-sync/bindings/binding-p9-e2e-1');
    await expect(page.getByText('productionReady=false')).toBeVisible({ timeout: 30000 });
    await expect(page.getByText(/binding-p9/).first()).toBeVisible();
    await expect(page.getByText('token')).toHaveCount(0);
    await expectNoRootOverflow(page);
  });
});
