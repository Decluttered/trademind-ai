import type { BrowserManager } from '../browser/manager.js';
import type { CollectProviderMeta } from '../types/provider-meta.js';
import type { NormalizedProduct } from '../types/product.js';

/** Input parameters for a single collection run */
export type CollectInput = {
  url: string;
  /** Provider-specific payload (e.g. custom: rule delivered by the backend) */
  options?: Record<string, unknown>;
};

/**
 * CollectorProvider — one implementation per collection source (1688, Taobao...); platform logic must not be hardcoded at the framework layer.
 */
export interface CollectorProvider {
  /** Matches the unified output field `source`, e.g. `1688` */
  readonly sourceId: string;

  /** Product metadata (drives the registry for /v1/providers) */
  readonly meta: CollectProviderMeta;

  /** Whether this URL is accepted (for quick validation without opening a browser) */
  canHandle(url: string): boolean;

  /** Runs the collection: BrowserManager supplies a temporary page instance */
  collect(browser: BrowserManager, input: CollectInput): Promise<NormalizedProduct>;
}
