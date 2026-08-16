import { describe, expect, it } from 'vitest';
import { amazonDECanHandle } from '../index.js';
import { assembleAmazonProduct, parseEuroCents } from '../parser.js';

describe('Amazon.de collector fixture parser', () => {
  it.each([['12,99 €', 1299], ['1.234,50 EUR', 123450], ['29 €', 2900]])('parses %s as integer cents', (raw, cents) => {
    expect(parseEuroCents(raw)).toBe(cents);
  });
  it('normalizes the fixture without persistence concerns', () => {
    const product = assembleAmazonProduct('https://www.amazon.de/dp/B012345678', {
      asin: 'b012345678', title: 'Fixture product', priceText: '19,95 €', availability: 'Auf Lager', images: ['https://images.example/item.jpg'],
      attributes: { Marke: 'Fixture' }, variants: [{ Farbe: 'Blau' }],
    });
    expect(product).toMatchObject({ source: 'amazon.de', sourceProductId: 'B012345678', priceCents: 1995, currency: 'EUR' });
    expect(product.raw.amazon).toBeDefined();
  });
  it('accepts product URLs only', () => {
    expect(amazonDECanHandle('https://www.amazon.de/dp/B012345678')).toBe(true);
    expect(amazonDECanHandle('https://www.amazon.com/dp/B012345678')).toBe(false);
    expect(amazonDECanHandle('https://www.amazon.de/s?k=test')).toBe(false);
  });
});
