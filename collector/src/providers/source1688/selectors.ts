/**
 * Multiple 1688 detail-page skins coexist, so each field has several fallback selector groups.
 * All are "best effort" matches; when none hit, script JSON and og:image fill in the gaps.
 */
export const TITLE_SELECTORS = [
  'h1.d-title',
  '.offer-title .title-text',
  '.title-content h1',
  '[class*="offer-title"] h1',
  'h1[class*="title"]',
];

/** Main image preview area (not the detail body); avoids overly broad [class*="gallery"] to prevent picking up service icons */
export const MAIN_GALLERY_SELECTORS = [
  '.vertical-img img',
  '.dot-img-footer-list img',
  '.tab-content-wrapper img',
  '.detail-gallery-preview img',
  '.detail-gallery-turn img',
  '.detail-gallery img',
  '[class*="offer-gallery"] img',
  '[class*="main-image"] img',
  '.swiper-slide img',
  '.obj-sku-img-item img',
  '.obj-header-image img',
];

/** Detail description area (usually a rich-text container) */
export const DETAIL_SELECTORS = [
  '#offer-template-0 img',
  '.offer-description img',
  '.offer-detail img',
  '.detail-desc-module img',
  '[class*="detail-description"] img',
  '[class*="offerDesc"] img',
  '[module-title="商品详情"] img',
  '.wireless-description img',
];

/** Parameters / attributes table */
export const ATTRIBUTE_ROW_SELECTORS = [
  '.offer-attrprogram .de-feature-item',
  '.offer-attr-item',
  '[class*="param-table"] tr',
  '.obj-content-table tr',
  '.offer-params tr',
  '#productAttributes .obj-content-table tr',
  '[module-title="商品属性"] tr',
];

/** SKU spec area (color/size, etc.) */
export const SKU_SECTION_SELECTORS = [
  '[class*="sku-item-wrapper"]',
  '[class*="sku-selector"]',
  '[class*="obj-sku"]',
  '[class*="sale-prop"]',
  '[class*="spec-item"]',
  '[class*="prop-item"]',
  '.module-od-sku-selection',
];

/** SKU size/stock table rows */
export const SKU_TABLE_ROW_SELECTORS = [
  '[class*="sku-table"] tr',
  '[class*="sku-item-list"] [class*="item"]',
  '[class*="table-sku"] tr',
  '.obj-sku .table tr',
];
