import { test, expect } from '../fixtures/admin.fixture';
import { expectAccountInTopNavbar, expectNoRootOverflow } from '../utils/assertions';

const smokeRoutes = [
  { path: '/dashboard/product-operations', name: /运营总览|工作台/ },
  { path: '/collect/hub', name: /采集中心/ },
  { path: '/ai/operation-workbench', name: /商品运营工作台/ },
  { path: '/product/drafts', name: /商品草稿|E2E 商品草稿/ },
  { path: '/inventory/overview', name: /库存中心/ },
  { path: '/ops/task-center/alerts', name: /告警中心/ },
  { path: '/files', name: /文件管理/ },
];

test.describe('@smoke Admin route smoke', () => {
  for (const route of smokeRoutes) {
    test(`renders ${route.path} without login, fatal error, or writes`, async ({ admin, page }) => {
      await admin.goto(route.path);
      await expect(page.locator('#root')).toBeVisible();
      await expect(page.getByText(route.name).first()).toBeVisible();
      await expect(page).not.toHaveURL(/\/user\/login/);
      await expectAccountInTopNavbar(page);
      await expectNoRootOverflow(page);
      await admin.writeGuard.expectRequestCount('unexpected', 0);
    });
  }

  test('opens logout from the account dropdown without triggering a write', async ({ admin, page }) => {
    await admin.goto('/dashboard/product-operations');

    const accountTrigger = page.getByRole('button', { name: /^当前用户 / });
    await expect(accountTrigger).toHaveAttribute('aria-haspopup', 'menu');
    await expect(accountTrigger).toHaveAttribute('aria-expanded', 'false');

    await accountTrigger.click();
    await expect(accountTrigger).toHaveAttribute('aria-expanded', 'true');
    await expect(page.getByRole('menuitem', { name: /退出登录/ })).toBeVisible();
    await expect(page.getByText('返回登录页面')).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(accountTrigger).toHaveAttribute('aria-expanded', 'false');
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });
});
