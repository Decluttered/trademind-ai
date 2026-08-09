import { test, expect } from '../fixtures/admin.fixture';
import { E2E_PRODUCT_ID } from '../mocks/product.fixture';
import {
  expectAccountInTopNavbar,
  expectHeaderContentAligned,
  expectNoRootOverflow,
  expectPageChromeScrollbarsHidden,
  expectPageContentGuttersWithin,
  expectTopNavbarScrollBehavior,
} from '../utils/assertions';
import { layoutTokens } from '../../src/constants/layoutTokens';

const viewports = [
  { width: 1440, height: 900 },
  { width: 1280, height: 800 },
  { width: 1024, height: 768 },
  { width: 768, height: 900 },
  { width: 375, height: 812 },
];

const pages = [
  {
    path: `/product/drafts/${E2E_PRODUCT_ID}?tab=publish`,
    label: /刊登|商品详情/,
    verifyStickyNavigation: true,
  },
  { path: '/ops/task-center/alerts', label: /告警中心/ },
  { path: '/product/drafts', label: /商品草稿|E2E 商品草稿/ },
];

test.describe('@product-draft @responsive 五档响应式', () => {
  for (const viewport of viewports) {
    for (const target of pages) {
      test(`${target.path} has no root overflow at ${viewport.width}x${viewport.height}`, async ({ admin, page }) => {
        await page.setViewportSize(viewport);
        await admin.goto(target.path);
        await expect(page.getByText(target.label).first()).toBeVisible();
        await expectAccountInTopNavbar(page);
        await expectNoRootOverflow(page);
        await expectHeaderContentAligned(page);
        if (target.verifyStickyNavigation) {
          await expectTopNavbarScrollBehavior(page);
        }
        await expect(page.locator('#root')).toBeVisible();
      });
    }
  }

  test('global content track keeps wide-screen gutters compact', async ({ admin, page }) => {
    await page.setViewportSize({ width: 2048, height: 1024 });
    await admin.goto('/dashboard/product-operations');
    await expect(page.getByText(/运营总览|工作台/).first()).toBeVisible();
    await expectNoRootOverflow(page);
    await expectHeaderContentAligned(page);
    await expectPageChromeScrollbarsHidden(page);
    await expectPageContentGuttersWithin(
      page,
      layoutTokens.pageMaxOuterGap + layoutTokens.pagePaddingX,
    );
  });
});
