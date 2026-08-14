import { test, expect } from '../fixtures/admin.fixture';
import { ok, fail } from '../mocks/envelope';
import { e2eReadinessFailed, e2eReadinessPassed, E2E_PRODUCT_ID } from '../mocks/product.fixture';
import { E2E_SHOPEE_SHOP_ID } from '../mocks/publish';
import { e2eUser } from '../mocks/auth';
import { expectRequestCount } from '../utils/assertions';
import { PERMISSIONS } from '../../src/utils/permission';

const isProductionE2E = !!process.env.CI;

async function openPublish(page: import('@playwright/test').Page) {
  await page.goto(`/product/drafts/${E2E_PRODUCT_ID}?tab=publish`);
  await expect(page.getByRole('tab', { name: /刊登/ })).toHaveAttribute('aria-selected', 'true');
}

async function selectFirstMultiPlatformTarget(page: import('@playwright/test').Page) {
  await page.getByRole('checkbox', { name: /E2E Shopee 测试店铺/ }).check();
}

async function selectLegacyPublishShop(page: import('@playwright/test').Page) {
  await page.locator('.product-draft-publish__legacy-form').getByRole('combobox').click();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  await expect(page.locator('.product-draft-publish__legacy-form')).toContainText('E2E Shopee 测试店铺');
}

async function cancelSensitiveDialog(page: import('@playwright/test').Page, title: string) {
  const dialog = page.getByRole('dialog', { name: title });
  await dialog.getByRole('button', { name: /取\s*消|取消/ }).click();
}

async function submitLegacyPublish(page: import('@playwright/test').Page) {
  await page.locator('.product-draft-publish__legacy-form').getByRole('button', { name: '提交刊登', exact: true }).click();
}

async function confirmLegacyPublish(page: import('@playwright/test').Page) {
  await page.getByRole('dialog', { name: '确认提交刊登？' }).getByRole('button', { name: '确认提交刊登' }).click();
}

async function closeModal(page: import('@playwright/test').Page, title: string) {
  const dialog = page.getByRole('dialog', { name: title });
  await dialog.locator('button').last().click();
}

test.describe('@publish-safety 发布写请求安全', () => {
  test('creates multi-platform drafts with exact payload and no other writes', async ({ admin, page }) => {
    admin.writeGuard.allow({
      operation: 'create-multi-platform-drafts',
      method: 'POST',
      path: new RegExp(`/api/v1/products/${E2E_PRODUCT_ID}/publish-targets/create-drafts$`),
      response: ok({ batchId: 'e2e-batch', status: 'done', statusLabel: '已完成', targetCount: 1, successCount: 1, failedCount: 0, skippedCount: 0, targets: [] }),
    });
    await openPublish(page);
    await expect(page.getByRole('checkbox', { name: /E2E 抖店测试店铺/ })).toBeDisabled();
    await selectFirstMultiPlatformTarget(page);
    await page.getByRole('button', { name: '创建刊登草稿' }).click();
    await expectRequestCount(admin.writeGuard, 'create-multi-platform-drafts', 1);
    expect(admin.writeGuard.calls('create-multi-platform-drafts')[0].postDataJSON).toEqual({
      targets: [{ platform: 'shopee', shopId: E2E_SHOPEE_SHOP_ID }],
      onlyReady: false,
      retryFailedOnly: false,
    });
    expect(admin.writeGuard.allCalls().map((call) => call.path)).not.toContain(`/api/v1/products/${E2E_PRODUCT_ID}/publish`);
  });

  test('routes douyin draft creation to operation task review without a platform write', async ({ admin, page }) => {
    await openPublish(page);
    await page.getByRole('button', { name: '进入运营任务审核' }).click();
    await page.waitForURL((url) => url.pathname === '/ops/task-center/operation-tasks');
    expect(admin.writeGuard.allCalls()).toHaveLength(0);
    expect(admin.writeGuard.allCalls().map((call) => call.path)).not.toContain(`/api/v1/products/${E2E_PRODUCT_ID}/platform-configs/douyin_shop/create-draft`);
  });

  test('traditional publish is hidden in production and remains guarded in development', async ({ admin, page }) => {
    admin.writeGuard.allow({
      operation: 'publish-product',
      method: 'POST',
      path: new RegExp(`/api/v1/products/${E2E_PRODUCT_ID}/publish$`),
      response: ok({ id: 'e2e-publish-task', productId: E2E_PRODUCT_ID, shopId: E2E_SHOPEE_SHOP_ID, platform: 'shopee', taskType: 'publish', status: 'queued', mode: 'publish', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }),
    });
    await openPublish(page);
    if (isProductionE2E) {
      await expect(page.locator('.product-draft-publish__legacy-form')).toHaveCount(0);
      await expect(page.getByText('传统提交刊登', { exact: true })).toHaveCount(0);
      await expectRequestCount(admin.writeGuard, 'publish-product', 0);
      return;
    }
    await selectLegacyPublishShop(page);

    await submitLegacyPublish(page);
    await cancelSensitiveDialog(page, '确认提交刊登？');
    await expectRequestCount(admin.writeGuard, 'publish-product', 0);

    await page.route(`**/api/v1/products/${E2E_PRODUCT_ID}/readiness**`, async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ok(e2eReadinessFailed)) }));
    await submitLegacyPublish(page);
    await confirmLegacyPublish(page);
    await expect(page.getByRole('dialog', { name: '发布检查未通过' })).toBeVisible();
    await closeModal(page, '发布检查未通过');
    await expectRequestCount(admin.writeGuard, 'publish-product', 0);

    await page.route(`**/api/v1/products/${E2E_PRODUCT_ID}/readiness**`, async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ok(e2eReadinessPassed)) }));
    await openPublish(page);
    await selectLegacyPublishShop(page);
    await submitLegacyPublish(page);
    await confirmLegacyPublish(page);
    await expectRequestCount(admin.writeGuard, 'publish-product', 1);
    expect(admin.writeGuard.calls('publish-product')[0].postDataJSON).toEqual({ shopId: E2E_SHOPEE_SHOP_ID, options: {} });

    await page.route(`**/api/v1/products/${E2E_PRODUCT_ID}/publish`, async (route) => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fail('e2e publish failed', 50001, null)) }));
  });

  test('legacy traditional task retry is hidden in production', async ({ admin, page }) => {
    await page.route('**/api/v1/product-publish/tasks**', async (route) => {
      const url = new URL(route.request().url());
      if (url.pathname !== '/api/v1/product-publish/tasks') {
        await route.fallback();
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(ok({
          list: [{
            id: 'e2e-legacy-publish-task',
            productId: E2E_PRODUCT_ID,
            shopId: E2E_SHOPEE_SHOP_ID,
            shopName: 'E2E Shopee 测试店铺',
            productTitle: 'E2E 旧传统刊登任务',
            platform: 'shopee',
            taskType: 'product_publish',
            status: 'failed',
            publishStatus: 'failed',
            mode: 'publish',
            retryable: true,
            errorCode: 'LEGACY_FAILURE',
            errorMessage: '旧任务失败',
            createdAt: '2026-01-01T00:00:00Z',
            updatedAt: '2026-01-01T00:01:00Z',
          }],
          pagination: { page: 1, pageSize: 20, total: 1, totalPages: 1 },
        })),
      });
    });

    await page.goto('/product/publish-tasks?tab=tasks');
    const row = page.getByRole('row', { name: /E2E 旧传统刊登任务/ });
    await expect(row).toBeVisible();
    if (isProductionE2E) {
      await expect(row.getByRole('button', { name: '重试' })).toHaveCount(0);
    } else {
      await expect(row.getByRole('button', { name: '重试' })).toBeVisible();
    }
    expect(admin.writeGuard.allCalls()).toHaveLength(0);
  });

  for (const scenario of [
    { role: 'readonly', permissions: [PERMISSIONS.PRODUCT_VIEW] },
    {
      role: 'reviewer',
      permissions: [
        PERMISSIONS.PRODUCT_VIEW,
        PERMISSIONS.OPERATION_TASK_AUDIT_READ,
        PERMISSIONS.OPERATION_TASK_REVIEW,
        PERMISSIONS.OPERATION_TASK_EXECUTE,
        PERMISSIONS.OPERATION_TASK_RETRY,
      ],
    },
  ]) {
    test(`${scenario.role} cannot select targets or trigger publish writes`, async ({ admin, page }) => {
      await page.route('**/api/v1/auth/profile', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(ok({ ...e2eUser, role: scenario.role, permissions: scenario.permissions })),
        });
      });
      await page.route('**/api/v1/product-publish/batches/e2e-batch-permission', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(ok({
            id: 'e2e-batch-permission',
            batchType: 'multi_product',
            name: 'E2E 权限批次',
            status: 'partial_success',
            statusLabel: '部分成功',
            productCount: 1,
            targetCount: 2,
            taskCount: 2,
            successCount: 0,
            failedCount: 1,
            skippedCount: 1,
            createdAt: '2026-01-01T00:00:00Z',
            finishedAt: '2026-01-01T00:01:00Z',
            items: [],
          })),
        });
      });

      await openPublish(page);
      await expect(page.getByText('当前账号无刊登草稿写权限').first()).toBeVisible();
      await expect(page.getByRole('checkbox', { name: /E2E Shopee 测试店铺/ })).toBeDisabled();
      await expect(page.getByRole('button', { name: '检查所选目标' })).toBeDisabled();
      await expect(page.getByRole('button', { name: '创建刊登草稿' })).toBeDisabled();
      await expect(page.getByRole('button', { name: '只处理可以创建草稿的目标' })).toBeDisabled();
      await expect(page.getByRole('button', { name: '进入运营任务审核' })).toBeDisabled();
      if (isProductionE2E) {
        await expect(page.locator('.product-draft-publish__legacy-form')).toHaveCount(0);
      } else {
        await expect(page.locator('.product-draft-publish__legacy-form').getByRole('combobox')).toBeDisabled();
        await expect(page.locator('.product-draft-publish__legacy-form').getByRole('button', { name: '提交刊登' })).toBeDisabled();
      }

      await page.goto('/product/publish-batches/e2e-batch-permission');
      await expect(page.getByRole('button', { name: '重试失败项' })).toBeDisabled();
      await expect(page.getByRole('button', { name: '取消等待项' })).toBeDisabled();
      expect(admin.writeGuard.allCalls()).toHaveLength(0);
    });
  }
});
