import { test, expect } from '../fixtures/admin.fixture';
import { e2eUser } from '../mocks/auth';
import { fail, ok } from '../mocks/envelope';
import { expectNoRootOverflow } from '../utils/assertions';

test.describe('@alert-center unified production workflow', () => {
  test('restores both alert sources from the URL and keeps refresh stable', async ({ admin, page }) => {
    await admin.goto('/ops/task-center/alerts?source=system');
    await expect(page.getByRole('tab', { name: '系统告警' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByText('E2E API error rate alert')).toBeVisible();

    await page.reload();
    await expect(page).toHaveURL(/source=system/);
    await expect(page.getByRole('tab', { name: '系统告警' })).toHaveAttribute('aria-selected', 'true');

    await page.getByRole('tab', { name: '业务告警' }).click();
    await expect(page).toHaveURL(/source=business/);
    await page.reload();
    await expect(page.getByRole('tab', { name: '业务告警' })).toHaveAttribute('aria-selected', 'true');
  });

  test('confirms one acknowledge write and sends no write when cancelled', async ({ admin, page }) => {
    admin.writeGuard.allow({
      operation: 'ack-alert',
      method: 'POST',
      path: /\/api\/v1\/observability\/alerts\/alert-e2e-1\/ack$/,
      response: { code: 0, message: 'ok', data: { id: 'alert-e2e-1', status: 'acknowledged' } },
    });
    await admin.goto('/ops/task-center/alerts?source=system');

    await page.getByRole('button', { name: '确认' }).first().click();
    const cancelDialog = page.getByRole('dialog', { name: '确认系统告警' });
    await cancelDialog.getByRole('button', { name: /取\s*消/ }).click();
    await admin.writeGuard.expectRequestCount('ack-alert', 0);

    await page.getByRole('button', { name: '确认' }).first().click();
    const confirmDialog = page.getByRole('dialog', { name: '确认系统告警' });
    await confirmDialog.getByRole('button', { name: '确认告警' }).dblclick();
    await admin.writeGuard.expectRequestCount('ack-alert', 1);
  });

  test('requires a silence reason and sends the bounded duration once', async ({ admin, page }) => {
    admin.writeGuard.allow({
      operation: 'silence-alert',
      method: 'POST',
      path: /\/api\/v1\/observability\/alerts\/alert-e2e-1\/silence$/,
      response: {
        code: 0,
        message: 'ok',
        data: { id: 'alert-e2e-1', status: 'silenced', expiresAt: '2026-08-10T16:30:00Z' },
      },
    });
    await admin.goto('/ops/task-center/alerts?source=system');

    await page.getByRole('button', { name: '静默' }).first().click();
    const dialog = page.getByRole('dialog', { name: '静默系统告警' });
    await dialog.getByRole('textbox', { name: '静默原因' }).fill('计划维护窗口');
    await dialog.getByRole('spinbutton', { name: '静默时长（小时）' }).fill('8');
    await dialog.getByRole('button', { name: '确认静默' }).dblclick();

    await admin.writeGuard.expectRequestCount('silence-alert', 1);
    expect(admin.writeGuard.calls('silence-alert')[0]?.postDataJSON).toEqual({
      reason: '计划维护窗口',
      durationHours: 8,
    });
  });

  test('shows a recoverable error when system alerts cannot be loaded', async ({ admin, page }) => {
    await page.route('**/api/v1/observability/alerts**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(fail('alert service unavailable')),
      });
    });

    await admin.goto('/ops/task-center/alerts?source=system');
    await expect(page.getByText('系统告警加载失败')).toBeVisible();
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });

  test('shows the system alert empty state without writes', async ({ admin, page }) => {
    await page.route('**/api/v1/observability/alerts**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          ok({
            items: [],
            pagination: { page: 1, pageSize: 20, total: 0, totalPages: 0 },
          }),
        ),
      });
    });

    await admin.goto('/ops/task-center/alerts?source=system');
    await expect(page.getByText('暂无系统告警')).toBeVisible();
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });

  test('keeps system alerts read-only without mutation permissions', async ({ admin, page }) => {
    await page.route('**/api/v1/auth/profile', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          ok({
            ...e2eUser,
            role: 'readonly',
            permissions: ['alerts.read'],
          }),
        ),
      });
    });

    await admin.goto('/ops/task-center/alerts?source=system');
    await expect(page.getByText('当前账号可查看系统告警，处置操作需要单独授权')).toBeVisible();
    await expect(page.getByRole('button', { name: '确认' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '静默' })).toHaveCount(0);
    await admin.writeGuard.expectRequestCount('unexpected', 0);
  });

  for (const viewport of [
    { width: 1440, height: 900 },
    { width: 375, height: 812 },
  ]) {
    test(`keeps the system alert view within the root at ${viewport.width}px`, async ({ admin, page }) => {
      await page.setViewportSize(viewport);
      await admin.goto('/ops/task-center/alerts?source=system');
      await expect(page.getByText('E2E API error rate alert')).toBeVisible();
      await expectNoRootOverflow(page);
    });
  }
});
