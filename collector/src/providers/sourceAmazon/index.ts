import type { Page } from 'playwright';
import { getDefaultNavigationTimeoutMs } from '../../config/env.js';
import type { BrowserManager } from '../../browser/manager.js';
import type { CollectorProvider } from '../collector-provider.js';
import type { NormalizedProduct } from '../../types/product.js';
import type { CollectFeature } from '../../types/provider-meta.js';
import { assembleAmazonProduct, type AmazonPagePayload } from './parser.js';

export function amazonDECanHandle(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.protocol === 'https:' && (parsed.hostname === 'amazon.de' || parsed.hostname.endsWith('.amazon.de')) &&
      (/\/dp\/[A-Z0-9]{10}/i.test(parsed.pathname) || /\/gp\/product\/[A-Z0-9]{10}/i.test(parsed.pathname));
  } catch { return false; }
}

async function extract(page: Page): Promise<AmazonPagePayload> {
  return page.evaluate(() => {
    const text = (selector: string) => document.querySelector(selector)?.textContent?.trim() || '';
    const attr = (selector: string, name: string) => document.querySelector(selector)?.getAttribute(name) || '';
    const asin = (document.querySelector<HTMLInputElement>('#ASIN')?.value ||
      document.body?.getAttribute('data-asin') || location.pathname.match(/\/(?:dp|gp\/product)\/([A-Z0-9]{10})/i)?.[1] || '').toUpperCase();
    const attributes: Record<string, string> = {};
    document.querySelectorAll('#productOverview_feature_div tr, #productDetails_detailBullets_sections1 tr').forEach((row) => {
      const cells = Array.from(row.querySelectorAll('th,td')).map((cell) => cell.textContent?.trim() || '').filter(Boolean);
      if (cells.length >= 2) attributes[cells[0]] = cells.slice(1).join(' ');
    });
    document.querySelectorAll('#feature-bullets li span.a-list-item').forEach((node, index) => {
      const value = node.textContent?.trim(); if (value) attributes[`bullet_${index + 1}`] = value;
    });
    const images = new Set<string>();
    document.querySelectorAll<HTMLImageElement>('#altImages img, #landingImage').forEach((img) => {
      const dynamic = img.getAttribute('data-a-dynamic-image');
      if (dynamic) { try { Object.keys(JSON.parse(dynamic)).forEach((url) => images.add(url)); } catch { /* ignore page data */ } }
      const src = img.currentSrc || img.src; if (src) images.add(src.replace(/\._[^.]+_\./, '.'));
    });
    const gtinKey = Object.keys(attributes).find((key) => /GTIN|EAN|UPC/i.test(key));
    return {
      asin,
      canonicalUrl: attr('link[rel="canonical"]', 'href'),
      title: text('#productTitle'),
      priceText: text('#corePrice_feature_div .a-offscreen') || text('.a-price .a-offscreen'),
      availability: text('#availability'),
      brand: text('#bylineInfo').replace(/^Marke:\s*/i, ''),
      gtin: gtinKey ? attributes[gtinKey] : '',
      images: [...images], attributes, variants: [],
      challenge: /captcha|robot check|geben sie die zeichen/i.test(`${document.title} ${document.body?.innerText.slice(0, 500)}`),
    };
  });
}

export const amazonDECollectorProvider: CollectorProvider = {
  sourceId: 'amazon.de',
  meta: {
    name: 'Amazon.de Produkt-Collector', description: 'Erfasst Produktdaten und ASIN-Snapshots von Amazon.de.',
    status: 'beta', batchSupported: false,
    urlPatterns: ['https://www.amazon.de/dp/*', 'https://www.amazon.de/gp/product/*'],
    features: ['title', 'price', 'mainImages', 'attributes'] satisfies CollectFeature[],
    notes: 'Nur öffentliche Amazon.de-Produktseiten; Login- und Challenge-Seiten werden abgelehnt.',
  },
  canHandle: amazonDECanHandle,
  async collect(browser: BrowserManager, input): Promise<NormalizedProduct> {
    if (!amazonDECanHandle(input.url)) throw new Error('INVALID_URL:not_an_amazon_de_product_url');
    return browser.withPageLocale('de-DE', async (page) => {
      const timeout = getDefaultNavigationTimeoutMs();
      try { await page.goto(input.url, { waitUntil: 'domcontentloaded', timeout }); }
      catch (error) { throw new Error(`NAVIGATION_FAILED:${error instanceof Error ? error.message : String(error)}`); }
      if (!amazonDECanHandle(page.url())) throw new Error('INVALID_URL:left_amazon_de_product_after_navigation');
      await page.waitForSelector('body', { timeout: Math.min(timeout, 8000) });
      return assembleAmazonProduct(input.url, await extract(page));
    });
  },
};
