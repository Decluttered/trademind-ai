import type { NormalizedProduct } from '../../types/product.js';

export type AmazonPagePayload = {
  asin?: string;
  canonicalUrl?: string;
  title?: string;
  priceText?: string;
  availability?: string;
  brand?: string;
  gtin?: string;
  images?: string[];
  attributes?: Record<string, string>;
  variants?: Array<Record<string, string>>;
  challenge?: boolean;
};

export function parseEuroCents(value: string | undefined): number | undefined {
  const raw = String(value ?? '').replace(/\u00a0/g, ' ').trim();
  if (!raw) return undefined;
  const match = raw.match(/(\d[\d.\s]*)(?:,(\d{1,2}))?\s*(?:€|EUR)?/i);
  if (!match) return undefined;
  const euros = Number(match[1].replace(/[.\s]/g, ''));
  const cents = Number((match[2] ?? '').padEnd(2, '0'));
  if (!Number.isSafeInteger(euros) || !Number.isInteger(cents)) return undefined;
  return euros * 100 + cents;
}

export function normalizeASIN(value: string | undefined): string {
  const asin = String(value ?? '').trim().toUpperCase();
  return /^[A-Z0-9]{10}$/.test(asin) ? asin : '';
}

export function assembleAmazonProduct(requestedUrl: string, payload: AmazonPagePayload): NormalizedProduct {
  const asin = normalizeASIN(payload.asin);
  const title = String(payload.title ?? '').trim();
  if (payload.challenge && !title) throw new Error('PAGE_BLOCKED_OR_VERIFY_REQUIRED:amazon_challenge');
  if (!asin) throw new Error('COLLECT_FAILED:missing_or_invalid_asin');
  if (!title) throw new Error('COLLECT_FAILED:missing_product_title');
  const priceCents = parseEuroCents(payload.priceText);
  return {
    source: 'amazon.de',
    sourceUrl: payload.canonicalUrl || requestedUrl,
    sourceProductId: asin,
    title,
    currency: 'EUR',
    priceCents,
    availability: String(payload.availability ?? '').trim() || undefined,
    brand: String(payload.brand ?? '').trim() || undefined,
    gtin: String(payload.gtin ?? '').trim() || undefined,
    variants: payload.variants ?? [],
    mainImages: [...new Set((payload.images ?? []).filter((x) => /^https:\/\//i.test(x)))],
    descriptionImages: [],
    attributes: payload.attributes ?? {},
    skus: [],
    raw: { amazon: payload },
  };
}
