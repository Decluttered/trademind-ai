/**
 * Unified product structure agreed with Go / the main business logic (any collection source is ultimately normalized to this format).
 */
export type NormalizedProduct = {
  source: string;
  sourceUrl: string;
  title: string;
  currency: string;
  /** Source-native stable identifier, e.g. Amazon ASIN. */
  sourceProductId?: string;
  /** Integer minor units. Float prices are intentionally not part of the Phase-1 contract. */
  priceCents?: number;
  availability?: string;
  brand?: string;
  gtin?: string;
  variants?: Array<Record<string, string>>;
  /** Real text description from the page (not AI-generated) */
  mainDescription?: string;
  mainImages: string[];
  descriptionImages: string[];
  attributes: Record<string, string | number | boolean>;
  skus: ProductSku[];
  /** Raw snapshot of the platform page, required for later review and re-parsing */
  raw: Record<string, unknown>;
};

export type ProductSku = {
  id?: string;
  /** Key-value pairs such as color, size, etc. */
  properties?: Record<string, string>;
  price?: number;
  stock?: number;
  skuCode?: string;
  image?: string;
  /** Raw snapshot at SKU granularity (preserved in product_skus.raw_data when Go writes it to the DB) */
  raw?: Record<string, unknown>;
};
