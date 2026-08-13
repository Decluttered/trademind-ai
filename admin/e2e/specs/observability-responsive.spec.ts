import { test, expect } from '../fixtures/admin.fixture';
import { expectHeaderContentAligned, expectNoRootOverflow } from '../utils/assertions';

const viewports = [
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 900 },
  { width: 375, height: 812 },
] as const;

test.describe('@smoke @observability-responsive 可观测性中心', () => {
  for (const viewport of viewports) {
    test(`keeps operational status usable at ${viewport.width}x${viewport.height}`, async ({
      admin,
      page,
    }) => {
      await page.setViewportSize(viewport);
      await admin.goto('/ops/observability');

      await expect(page.locator('.tm-page-container').getByText('可观测性中心', { exact: true }).first()).toBeVisible();
      await expect(page.getByText('总体运行状态')).toBeVisible();
      await expect(page.getByText('活跃系统告警')).toBeVisible();
      await expect(page.getByText('告警与服务目标')).toBeVisible();
      await expect(page.getByText('指标与遥测')).toBeVisible();
      await expect(page.getByRole('button', { name: '刷新运行状态' })).toBeVisible();
      await expectHeaderContentAligned(page);
      await expectNoRootOverflow(page);
      await admin.writeGuard.expectRequestCount('unexpected', 0);
    });
  }
});
