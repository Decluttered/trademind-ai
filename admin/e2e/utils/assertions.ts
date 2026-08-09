import { expect, type Locator, type Page } from '@playwright/test';
import type { ConsoleGuard } from './console-guard';
import type { NetworkWriteGuard } from './network-guard';

export async function expectNoRootOverflow(page: Page) {
  const value = await page.evaluate(() => ({
    htmlScrollWidth: document.documentElement.scrollWidth,
    htmlClientWidth: document.documentElement.clientWidth,
    bodyScrollWidth: document.body.scrollWidth,
    bodyClientWidth: document.body.clientWidth,
  }));
  expect(value.htmlScrollWidth, `html overflow: ${JSON.stringify(value)}`).toBeLessThanOrEqual(value.htmlClientWidth + 1);
  expect(value.bodyScrollWidth, `body overflow: ${JSON.stringify(value)}`).toBeLessThanOrEqual(value.bodyClientWidth + 1);
}

export async function expectActiveTab(page: Page, tabName: string) {
  const tab = page.getByRole('tab', { name: new RegExp(tabName) });
  await expect(tab, `active tab ${tabName}`).toHaveAttribute('aria-selected', 'true');
}

export async function expectSectionVisible(page: Page, sectionId: string) {
  await expect(page.locator(`#${sectionId}`), `section ${sectionId}`).toBeVisible();
}

export async function expectModalWithinViewport(page: Page) {
  await expectOverlayWithinViewport(page.locator('.ant-modal:visible').first(), page, 'modal');
}

export async function expectDrawerWithinViewport(page: Page) {
  await expectOverlayWithinViewport(page.locator('.ant-drawer-content-wrapper:visible').first(), page, 'drawer');
}

export async function expectHeaderContentAligned(page: Page) {
  const value = await page.evaluate(() => {
    const header = document.querySelector('.tm-page-container .ant-page-header')?.getBoundingClientRect();
    const content = document.querySelector('.tm-page-container__content')?.getBoundingClientRect();
    if (!header || !content) return null;
    return {
      headerLeft: header.left,
      headerRight: header.right,
      contentLeft: content.left,
      contentRight: content.right,
      leftDelta: Math.abs(header.left - content.left),
      rightDelta: Math.abs(header.right - content.right),
    };
  });
  expect(value, 'header/content alignment metrics').not.toBeNull();
  if (!value) return;
  expect(value.leftDelta, `header/content left delta ${JSON.stringify(value)}`).toBeLessThanOrEqual(4);
  expect(value.rightDelta, `header/content right delta ${JSON.stringify(value)}`).toBeLessThanOrEqual(4);
}

export async function expectPageContentGuttersWithin(page: Page, maxGutter: number) {
  const value = await page.evaluate(() => {
    const shell = document.querySelector('.ant-pro-layout-content')?.getBoundingClientRect();
    const contentElement = document.querySelector<HTMLElement>('.tm-page-container__content');
    const content = contentElement?.getBoundingClientRect();
    if (!shell || !content || !contentElement) return null;
    const style = window.getComputedStyle(contentElement);
    return {
      shellLeft: shell.left,
      shellRight: shell.right,
      contentLeft: content.left,
      contentRight: content.right,
      leftGutter: content.left - shell.left + Number.parseFloat(style.paddingLeft || '0'),
      rightGutter: shell.right - content.right + Number.parseFloat(style.paddingRight || '0'),
    };
  });
  expect(value, 'page content gutter metrics').not.toBeNull();
  if (!value) return;
  expect(value.leftGutter, `page left gutter ${JSON.stringify(value)}`).toBeGreaterThanOrEqual(-1);
  expect(value.rightGutter, `page right gutter ${JSON.stringify(value)}`).toBeGreaterThanOrEqual(-1);
  expect(value.leftGutter, `page left gutter ${JSON.stringify(value)}`).toBeLessThanOrEqual(maxGutter + 1);
  expect(value.rightGutter, `page right gutter ${JSON.stringify(value)}`).toBeLessThanOrEqual(maxGutter + 1);
}

export async function expectPageChromeScrollbarsHidden(page: Page) {
  const value = await page.evaluate(() => {
    const layout = document.querySelector('.tm-app-layout');
    const sider = layout?.querySelector('.ant-pro-sider');
    const siderScroller = sider
      ? [sider, ...Array.from(sider.querySelectorAll('*'))].find((element) => {
          const overflowY = window.getComputedStyle(element).overflowY;
          return overflowY === 'auto' || overflowY === 'scroll';
        })
      : undefined;
    const pageScroller = document.scrollingElement as HTMLElement | null;
    const pageMaxScroll = pageScroller ? pageScroller.scrollHeight - pageScroller.clientHeight : 0;
    const pageScrollTopBefore = pageScroller?.scrollTop ?? 0;
    if (pageScroller && pageMaxScroll > 0) {
      pageScroller.scrollTop = Math.min(pageScrollTopBefore + 80, pageMaxScroll);
    }
    const pageScrollTopAfter = pageScroller?.scrollTop ?? 0;
    if (pageScroller) pageScroller.scrollTop = pageScrollTopBefore;

    const scrollbarState = (element: Element) => {
      const scrollbarWidth = window.getComputedStyle(element).getPropertyValue('scrollbar-width');
      const webkitDisplay = window.getComputedStyle(element, '::-webkit-scrollbar').display;

      return {
        scrollbarWidth,
        webkitDisplay,
        hidden: scrollbarWidth === 'none' || webkitDisplay === 'none',
      };
    };

    return {
      hasLayout: Boolean(layout),
      hasSiderScroller: Boolean(siderScroller),
      siderOverflowY: siderScroller ? window.getComputedStyle(siderScroller).overflowY : null,
      html: scrollbarState(document.documentElement),
      body: scrollbarState(document.body),
      sider: siderScroller ? scrollbarState(siderScroller) : null,
      pageMaxScroll,
      pageScrollTopBefore,
      pageScrollTopAfter,
    };
  });

  expect(value.hasLayout, `app layout ${JSON.stringify(value)}`).toBe(true);
  expect(value.hasSiderScroller, `sider scroll container ${JSON.stringify(value)}`).toBe(true);
  expect(value.siderOverflowY, `sider keeps scrolling ${JSON.stringify(value)}`).toBe('auto');
  expect(value.html.hidden, `html scrollbar ${JSON.stringify(value)}`).toBe(true);
  expect(value.body.hidden, `body scrollbar ${JSON.stringify(value)}`).toBe(true);
  expect(value.sider?.hidden, `sider scrollbar ${JSON.stringify(value)}`).toBe(true);
  expect(value.pageMaxScroll, `page remains scrollable ${JSON.stringify(value)}`).toBeGreaterThan(0);
  expect(value.pageScrollTopAfter, `page scroll movement ${JSON.stringify(value)}`).toBeGreaterThan(
    value.pageScrollTopBefore,
  );
}

export async function expectRequestCount(tracker: NetworkWriteGuard, operation: string, count: number) {
  await tracker.expectRequestCount(operation, count);
}

export async function expectNoUnexpectedWrites(tracker: NetworkWriteGuard) {
  await tracker.expectNoUnexpectedWrites();
}

export async function expectNoFatalConsoleErrors(consoleGuard: ConsoleGuard) {
  await consoleGuard.expectNoFatalErrors();
}

async function expectOverlayWithinViewport(locator: Locator, page: Page, label: string) {
  await expect(locator, `${label} visible`).toBeVisible();
  const box = await locator.boundingBox();
  const viewport = page.viewportSize();
  expect(box, `${label} bounding box`).not.toBeNull();
  expect(viewport, 'viewport size').not.toBeNull();
  if (!box || !viewport) return;
  expect(box.x, `${label} left`).toBeGreaterThanOrEqual(0);
  expect(box.y, `${label} top`).toBeGreaterThanOrEqual(0);
  expect(box.x + box.width, `${label} right`).toBeLessThanOrEqual(viewport.width + 1);
  expect(box.y + box.height, `${label} bottom`).toBeLessThanOrEqual(viewport.height + 1);
}
