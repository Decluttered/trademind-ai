/** Collector metadata and capability tags aligned with the admin panel / Go fallback list */

export type ProviderStatus = 'available' | 'beta' | 'planned' | 'disabled';

/** Structured capability enum (corresponds to server-side DB fields, no parsing logic change) */
export type CollectFeature = 'title' | 'price' | 'mainImages' | 'descriptionImages' | 'attributes' | 'skus';

export type CollectProviderMeta = {
  name: string;
  description: string;
  status: ProviderStatus;
  batchSupported: boolean;
  urlPatterns: string[];
  features: CollectFeature[];
  notes: string;
};

/** A single row of GET /v1/providers (includes source) */
export type CollectProviderPublic = {
  source: string;
} & CollectProviderMeta;
