import type { NormalizedProduct, ProductSku } from '../../types/product.js';

/** Spec dimensions extracted from the DOM on the browser side */
export type DomSkuDimension = {
  name: string;
  values: string[];
};

/** Rows extracted from the size table on the browser side (includes price/stock text) */
export type DomSkuTableRow = {
  label: string;
  priceText?: string;
  stockText?: string;
};

/** Raw extraction result returned by browser-side evaluate (serializable) */
export type BrowserExtractPayload = {
  finalUrl: string;
  docTitle: string;
  meta: {
    description?: string;
    ogTitle?: string;
    ogImage?: string;
    twitterImage?: string;
    keywords?: string;
  };
  headingText: string;
  galleryUrls: string[];
  detailUrls: string[];
  /** Text from the DOM price region */
  domPriceTexts: string[];
  paramPairs: Array<{ key: string; value: string }>;
  domSkuDimensions: DomSkuDimension[];
  domSkuTableRows: DomSkuTableRow[];
  /** Script fragments that may contain JSON (truncated, for re-parsing on the Node side) */
  scriptSnippets: string[];
};

export type Parse1688Result = Pick<
  NormalizedProduct,
  'title' | 'mainImages' | 'descriptionImages' | 'attributes' | 'skus'
> & {
  raw: Record<string, unknown>;
  collectStatus?: 'success' | 'partial_success' | 'failed';
  completeness?: number;
  missingFields?: string[];
  warnings?: string[];
  extractDebug?: Record<string, unknown>;
};

export type { ProductSku };
